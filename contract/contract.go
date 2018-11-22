package contract

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/zhigui-projects/zwasm/state"
	"github.com/zhigui-projects/zwasm/types"
)

var (
	errUnmarshalInitCall = errors.New("failed unmarshal init call info")
	errUnmarshalCall     = errors.New("failed unmarshal function call info")
	errNoContract        = errors.New("no contract found")
	errGasExceed         = errors.New("gas limit exceed")
)

type Context struct {
	gasLimit      uint64
	senderAddress []byte
}

func Create(crtState *state.ContractState, context *Context, code []byte) (int64, uint64, error) {
	contract, codeLen, deployGas, err := setCode(crtState, code, context.gasLimit)
	if err != nil {
		return 0, 0, err
	}

	crtState.SetData([]byte("Creator"), context.senderAddress)
	ci := &types.CallInfo{}
	if len(code) != int(codeLen) {
		err = proto.Unmarshal(code[codeLen:], ci)
		if err != nil {
			return 0, 0, errUnmarshalInitCall
		}
	}

	if ci == nil {
		return 0, deployGas, nil
	}

	ret, callGas, err := call(contract, ci, newExternalResolver(context, crtState))
	return ret, deployGas + callGas, err
}

func Call(crtState *state.ContractState, context *Context, code []byte) (int64, uint64, error) {
	contract := getCode(crtState, nil)

	if contract == nil {
		return 0, 0, errNoContract
	}

	var ci *types.CallInfo
	err := proto.Unmarshal(code, ci)
	if err != nil {
		return 0, 0, errUnmarshalCall
	}

	return call(contract, ci, newExternalResolver(context, crtState))
}
