package contract

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/zhigui-projects/zwasm/state"
	"github.com/zhigui-projects/zwasm/types"
)

var (
	errNoInitFunction    = errors.New("contract no init function")
	errUnmarshalInitCall = errors.New("failed unmarshal init call info")
	errUnmarshalCall     = errors.New("failed unmarshal function call info")
	errNoContract        = errors.New("no contract found")
)

type Context struct {
	gasLimit      uint64
	senderAddress []byte
}

func Create(crtState *state.ContractState, context *Context, code []byte) (int64, uint64, error) {
	contract, codeLen, err := setCode(crtState, code)
	if err != nil {
		return 0, 0, err
	}

	crtState.SetData([]byte("Creator"), context.senderAddress)
	var ci *types.CallInfo
	if len(code) != int(codeLen) {
		err = proto.Unmarshal(code[codeLen:], ci)
		if err != nil {
			return 0, 0, errUnmarshalInitCall
		}
	}

	if ci == nil {
		return 0, 0, errNoInitFunction
	}

	return call(contract, ci, newExternalResolver(context, crtState))
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
