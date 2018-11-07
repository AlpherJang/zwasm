package state

import (
	"bytes"
	"os"
	"testing"

	"github.com/aergoio/aergo-lib/db"
	"github.com/zhigui-projects/zwasm/common"
	"github.com/zhigui-projects/zwasm/types"
)

func TestContractStateCode(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()
	testAddress := []byte("test_address")
	testBytes := []byte("test_bytes")
	contractState, err := manager.OpenContractStateAccount(types.ToAccountID(testAddress, hashFunc))
	if err != nil {
		t.Errorf("counld not open contract state : %s", err.Error())
	}
	err = contractState.SetCode(testBytes)
	if err != nil {
		t.Errorf("counld set code to contract state : %s", err.Error())
	}
	res, err := contractState.GetCode()
	if !bytes.Equal(res, testBytes) {
		t.Errorf("different code detected : %s =/= %s", testBytes, string(res))
	}
}

func TestContractStateData(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()
	testAddress := []byte("test_address")
	testBytes := []byte("test_bytes")
	testKey := []byte("test_key")
	contractState, err := manager.OpenContractStateAccount(types.ToAccountID(testAddress, hashFunc))
	if err != nil {
		t.Errorf("counld not open contract state : %s", err.Error())
	}
	err = contractState.SetData(testKey, testBytes)
	if err != nil {
		t.Errorf("counld set data to contract state : %s", err.Error())
	}
	res, err := contractState.GetData(testKey)
	if !bytes.Equal(res, testBytes) {
		t.Errorf("different data detected : %s =/= %s", testBytes, string(res))
	}
	err = manager.CommitContractState(contractState)
	if err != nil {
		t.Errorf("counld commit contract state : %s", err.Error())
	}
}

func TestContractStateEmpty(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()
	testAddress := []byte("test_address")
	contractState, err := manager.OpenContractStateAccount(types.ToAccountID(testAddress, hashFunc))
	if err != nil {
		t.Errorf("counld not open contract state : %s", err.Error())
	}
	err = manager.CommitContractState(contractState)
	if err != nil {
		t.Errorf("counld commit contract state : %s", err.Error())
	}
}

func TestContractStateReOpenData(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()
	testAddress := []byte("test_address")
	testBytes := []byte("test_bytes")
	testKey := []byte("test_key")
	contractState, err := manager.OpenContractStateAccount(types.ToAccountID(testAddress, hashFunc))
	if err != nil {
		t.Errorf("counld not open contract state : %s", err.Error())
	}
	err = contractState.SetData(testKey, testBytes)
	if err != nil {
		t.Errorf("counld set data to contract state : %s", err.Error())
	}
	res, err := contractState.GetData(testKey)
	if err != nil {
		t.Errorf("counld set data to contract state : %s", err.Error())
	}
	if !bytes.Equal(res, testBytes) {
		t.Errorf("different data detected : %s =/= %s", testBytes, string(res))
	}
	err = manager.CommitContractState(contractState)
	if err != nil {
		t.Errorf("counld commit contract state : %s", err.Error())
	}
	//contractState2, err := chainStateDB.OpenContractStateAccount(types.ToAccountID(testAddress))
	contractState2, err := manager.OpenContractState(contractState.State)
	if err != nil {
		t.Errorf("counld not open contract state : %s", err.Error())
	}
	res2, err := contractState2.GetData(testKey)
	if err != nil {
		t.Errorf("counld not get contract state : %s", err.Error())
	}
	if !bytes.Equal(res2, testBytes) {
		t.Errorf("different data detected : %s =/= %s", testBytes, string(res2))
	}
}
