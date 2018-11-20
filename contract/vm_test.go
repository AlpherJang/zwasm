package contract

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/aergoio/aergo-lib/db"
	"github.com/stretchr/testify/assert"
	"github.com/zhigui-projects/zwasm/common"
	"github.com/zhigui-projects/zwasm/state"
	"github.com/zhigui-projects/zwasm/types"
)

func loadCode() ([]byte, error) {
	code, err := ioutil.ReadFile("fixture/contract.wasm")
	if err != nil {
		return nil, err
	}

	return code, nil
}

func createContractState(t *testing.T) (*state.ContractState, error) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := state.NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()
	testAddress := []byte(t.Name())
	return manager.OpenContractStateAccount(types.ToAccountID(testAddress, hashFunc))
}

func TestCall(t *testing.T) {
	code, err := loadCode()
	assert.NoError(t, err)
	ci := &types.CallInfo{Name: "invoke"}

	crtState, err := createContractState(t)
	context := &Context{gasLimit: 10000, senderAddress: []byte("sender")}
	_, usedGas, err := call(code, ci, newExternalResolver(context, crtState))
	assert.True(t, usedGas > 0)
	assert.NoError(t, err)

	val, err := crtState.GetData([]byte("abc"))
	val1, err := crtState.GetData([]byte("abc1"))
	assert.Equal(t, "xyz", string(val))
	assert.Equal(t, "xyz", string(val1))
}
