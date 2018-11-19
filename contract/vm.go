package contract

import (
	"encoding/binary"
	"fmt"

	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/exec"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/zhigui-projects/zwasm/state"
	"github.com/zhigui-projects/zwasm/types"
)

const defaultMemoryPages = 128
const defaultTableSize = 65536

var (
	errCreateVM            = errors.New("failed to create virtual machine")
	errNotSupportStartFunc = errors.New("not support start function")
	errDeployContract      = errors.New("cannot deploy contract")
	gasPolicy              = &compiler.SimpleGasPolicy{GasPerInstruction: 1}
)

type externalResolver struct {
	context  *Context
	crtState *state.ContractState
}

func newExternalResolver(context *Context, crtState *state.ContractState) *externalResolver {
	return &externalResolver{context: context, crtState: crtState}
}

func (shim *externalResolver) ResolveGlobal(module, field string) int64 {
	log.Debug().Msgf("Resolve global: %s %s\n", module, field)
	switch module {
	case "env":
		switch field {
		case "zwasm_magic":
			return 76
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

func (shim *externalResolver) ResolveFunc(module, field string) exec.FunctionImport {
	log.Debug().Msgf("Resolve func: %s %s\n", module, field)
	switch module {
	case "env":
		switch field {
		case "_get_len":
			return func(vm *exec.VirtualMachine) int64 {
				ptr := int(uint32(vm.GetCurrentFrame().Locals[0]))
				keyLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				key := vm.Memory[ptr : ptr+keyLen]

				value, err := shim.crtState.GetData(key)
				if err != nil {
					log.Error().Err(err)
					return -1
				}

				valueLen := len(value)
				return int64(valueLen)
			}
		case "_set":
			return func(vm *exec.VirtualMachine) int64 {
				keyPtr := int(uint32(vm.GetCurrentFrame().Locals[0]))
				keyLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				key := vm.Memory[keyPtr : keyPtr+keyLen]

				valuePtr := int(uint32(vm.GetCurrentFrame().Locals[2]))
				valueLen := int(uint32(vm.GetCurrentFrame().Locals[3]))
				value := vm.Memory[valuePtr : valuePtr+valueLen]

				err := shim.crtState.SetData(key, value)
				if err != nil {
					log.Error().Err(err)
					return -1
				} else {
					return 1
				}
			}
		case "_get":
			return func(vm *exec.VirtualMachine) int64 {
				keyPtr := int(uint32(vm.GetCurrentFrame().Locals[0]))
				keyLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				key := vm.Memory[keyPtr : keyPtr+keyLen]

				outValuePtr := int(uint32(vm.GetCurrentFrame().Locals[2]))
				value, err := shim.crtState.GetData(key)
				if err != nil {
					log.Error().Err(err)
					return -1
				} else {
					outValueMem := vm.Memory[outValuePtr : outValuePtr+len(value)]
					copy(outValueMem, value)
					return 1
				}
			}
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

func codeLength(val []byte) uint32 {
	return binary.LittleEndian.Uint32(val[0:])
}

func setCode(contractState *state.ContractState, code []byte) ([]byte, uint32, error) {
	if len(code) <= 4 {
		err := fmt.Errorf("invalid code (%d bytes is too short)", len(code))
		return nil, 0, err
	}
	codeLen := codeLength(code[0:])
	if uint32(len(code)) < codeLen {
		err := fmt.Errorf("invalid code (expected %d bytes, actual %d bytes)", codeLen, len(code))
		return nil, 0, err
	}
	sCode := code[4:codeLen]

	err := contractState.SetCode(sCode)
	if err != nil {
		return nil, 0, err
	}
	contract := getCode(contractState, sCode)
	if contract == nil {
		return nil, 0, errDeployContract
	}

	return contract, codeLen, nil
}

func getCode(contractState *state.ContractState, code []byte) []byte {
	var val []byte
	val = code
	if val == nil {
		var err error
		val, err = contractState.GetCode()

		if err != nil {
			return nil
		}
	}
	valLen := len(val)
	if valLen <= 4 {
		return nil
	}
	l := codeLength(val[0:])
	if 4+l > uint32(valLen) {
		return nil
	}
	return val[4 : 4+l]
}

func call(code []byte, callInfo *types.CallInfo, resolver *externalResolver) (int64, uint64, error) {
	vm, err := exec.NewVirtualMachine(code, exec.VMConfig{
		DefaultMemoryPages: defaultMemoryPages,
		DefaultTableSize:   defaultTableSize,
		GasLimit:           resolver.context.gasLimit,
	}, resolver, gasPolicy)

	if err != nil {
		return -1, 0, errCreateVM
	}

	if vm.Module.Base.Start != nil {
		return -1, 0, errNotSupportStartFunc
	}

	entryId, ok := vm.GetFunctionExport(callInfo.Name)
	if !ok {
		err = errors.Errorf("function %s not found", callInfo.Name)
		return -1, 0, err
	}

	argsLen := len(callInfo.Args)
	outArgsPtr := 0
	outArgsLen := 0
	if argsLen > 0 {
		outArgsPtr, outArgsLen = injectArgs(argsLen, outArgsPtr, vm, outArgsLen, callInfo)
	}

	ret, err := vm.Run(entryId, int64(outArgsPtr), int64(outArgsLen))
	if err != nil {
		return ret, vm.Gas, err
	}

	return ret, vm.Gas, nil
}

func injectArgs(argsLen int, outArgsPtr int, vm *exec.VirtualMachine, outArgsLen int, callInfo *types.CallInfo) (int, int) {
	argsLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(argsLenBytes, uint32(argsLen))
	outArgsPtr = len(vm.Memory)
	outArgsLen = outArgsLen + 4
	vm.Memory = append(vm.Memory, argsLenBytes...)
	for _, arg := range callInfo.Args {
		argLen := len(arg)
		argLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(argLenBytes, uint32(argLen))
		outArgsLen = outArgsLen + 4 + argLen
		vm.Memory = append(vm.Memory, argLenBytes...)
		vm.Memory = append(vm.Memory, arg...)
	}
	return outArgsPtr, outArgsLen
}
