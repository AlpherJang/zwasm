package contract

import (
	"encoding/binary"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aergoio/aergo-lib/db"
	"github.com/stretchr/testify/assert"
	"github.com/zhigui-projects/zwasm/common"
	"github.com/zhigui-projects/zwasm/state"
	"github.com/zhigui-projects/zwasm/types"
)

func TestCall(t *testing.T) {
	code, err := loadCode()
	assert.NoError(t, err)
	ci := &types.CallInfo{Name: "invoke"}

	store := createDB(t)
	defer closeDB(t, store)
	crtState, err := createContractState(t, store)
	context := &Context{gasLimit: 10000, senderAddress: []byte("sender")}
	_, usedGas, err := call(code, ci, newExternalResolver(context, crtState))
	assert.True(t, usedGas > 0)
	assert.NoError(t, err)

	val, err := crtState.GetData([]byte("abc"))
	val1, err := crtState.GetData([]byte("abc1"))
	assert.Equal(t, "xyz", string(val))
	assert.Equal(t, "xyz", string(val1))
}

func TestGetSetCode(t *testing.T) {
	code := []byte("abc")
	store := createDB(t)
	defer closeDB(t, store)
	crtState, err := createContractState(t, store)
	assert.NoError(t, err)
	ret := getCode(crtState, code)
	assert.Nil(t, ret)

	codeLen := len(code)
	codeLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(codeLenBytes, uint32(codeLen+4))
	codeLenBytes = append(codeLenBytes, code...)
	setCode(crtState, codeLenBytes)
	ret1 := getCode(crtState, nil)
	assert.Nil(t, ret1)

	crtState1, err := createContractState(t, store)
	correctCodeLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(correctCodeLenBytes, uint32(codeLen))
	correctCodeLenBytes = append(correctCodeLenBytes, code...)
	totalCodeLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(totalCodeLenBytes, uint32(4+4+codeLen))
	totalCodeLenBytes = append(totalCodeLenBytes, correctCodeLenBytes...)
	setCode(crtState1, totalCodeLenBytes)
	ret2 := getCode(crtState1, nil)
	assert.NotNil(t, ret2)
	assert.Equal(t, code, ret2)

	correctCodeLenBytes1 := make([]byte, 4)
	binary.LittleEndian.PutUint32(correctCodeLenBytes1, uint32(codeLen))
	correctCodeLenBytes1 = append(correctCodeLenBytes1, code...)
	ret3 := getCode(crtState, correctCodeLenBytes1)
	assert.NotNil(t, ret3)
	assert.Equal(t, code, ret3)
}

func loadCode() ([]byte, error) {
	code, err := ioutil.ReadFile("fixture/contract.wasm")
	if err != nil {
		return nil, err
	}

	return code, nil
}

func createDB(t *testing.T) db.DB {
	return db.NewDB(db.BadgerImpl, t.Name())
}

func createContractState(t *testing.T, store db.DB) (*state.ContractState, error) {
	hashFunc := common.HashFuncFactory("sha3")
	manager := state.NewManager(&store, nil, hashFunc)
	testAddress := []byte(t.Name())
	return manager.OpenContractStateAccount(types.ToAccountID(testAddress, hashFunc))
}

func closeDB(t *testing.T, store db.DB) {
	store.Close()
	os.RemoveAll(t.Name())
}
