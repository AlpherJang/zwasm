package state

import (
	"bytes"
	"testing"

	"encoding/hex"
	"os"

	"github.com/aergoio/aergo-lib/db"
	"github.com/stretchr/testify/assert"
	"github.com/zhigui-projects/zwasm/common"
	"github.com/zhigui-projects/zwasm/types"
)

var (
	testAccount = types.ToAccountID([]byte("test_address"), common.HashFuncFactory("sha3"))
	testRoot, _ = hex.DecodeString("04e9445fc3c13d9f6d8a98ae2a08c75a1a4fab5f8fd9e59b53ebdc91f2326cf0")
	testStates  = []types.State{
		{Nonce: 1, Balance: 100},
		{Nonce: 2, Balance: 200},
		{Nonce: 3, Balance: 300},
		{Nonce: 4, Balance: 400},
		{Nonce: 5, Balance: 500},
	}
	testSecondRoot, _ = hex.DecodeString("4e6c3c58dd76d350f751903db403a82e14b6800f2f90130ff95d8971b73b42ff")
	testSecondStates  = []types.State{
		{Nonce: 6, Balance: 600},
		{Nonce: 7, Balance: 700},
		{Nonce: 8, Balance: 800},
	}
)

func stateEquals(expected, actual *types.State) bool {
	return expected.Nonce == actual.Nonce &&
		expected.Balance == actual.Balance &&
		bytes.Equal(expected.CodeHash, actual.CodeHash) &&
		bytes.Equal(expected.StorageRoot, actual.StorageRoot)
}

func TestStateDBGetEmptyState(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// get nil state
	st, err := manager.GetState(testAccount)
	if err != nil {
		t.Errorf("failed to get state: %v", err.Error())
	}
	assert.Nil(t, st)

	// get empty state
	st, err = manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get account state: %v", err.Error())
	}
	assert.NotNil(t, st)
	assert.Empty(t, st)
}

func TestStateDBPutState(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// put state
	err := manager.PutState(testAccount, &testStates[0])
	if err != nil {
		t.Errorf("failed to put state: %v", err.Error())
	}

	// get state
	st, err := manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get account state: %v", err.Error())
	}
	assert.NotNil(t, st)
	assert.True(t, stateEquals(&testStates[0], st))
}

func TestStateDBRollback(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// put states
	initialRevision := manager.Snapshot()
	for _, v := range testStates {
		_ = manager.PutState(testAccount, &v)
	}
	revision := manager.Snapshot()
	for _, v := range testSecondStates {
		_ = manager.PutState(testAccount, &v)
	}

	// get state
	st, err := manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get account state: %v", err.Error())
	}
	assert.NotNil(t, st)
	assert.True(t, stateEquals(&testSecondStates[2], st))

	// rollback to snapshot
	err = manager.Rollback(revision)
	if err != nil {
		t.Errorf("failed to rollback: %v", err.Error())
	}
	st, err = manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get account state: %v", err.Error())
	}
	assert.NotNil(t, st)
	assert.True(t, stateEquals(&testStates[4], st))

	// rollback to initial revision snapshot
	err = manager.Rollback(initialRevision)
	if err != nil {
		t.Errorf("failed to rollback: %v", err.Error())
	}
	st, err = manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get account state: %v", err.Error())
	}
	assert.NotNil(t, st)
	assert.Empty(t, st)
}

func TestStateDBUpdateAndCommit(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	assert.Nil(t, manager.GetRoot())
	for _, v := range testStates {
		_ = manager.PutState(testAccount, &v)
	}
	assert.Nil(t, manager.GetRoot())

	err := manager.Update()
	if err != nil {
		t.Errorf("failed to update: %v", err.Error())
	}
	assert.NotNil(t, manager.GetRoot())
	assert.Equal(t, testRoot, manager.GetRoot())

	err = manager.Commit()
	if err != nil {
		t.Errorf("failed to commit: %v", err.Error())
	}
	assert.Equal(t, testRoot, manager.GetRoot())
}

func TestStateDBSetRoot(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// put states
	assert.Nil(t, manager.GetRoot())
	for _, v := range testStates {
		_ = manager.PutState(testAccount, &v)
	}
	_ = manager.Update()
	_ = manager.Commit()
	assert.Equal(t, testRoot, manager.GetRoot())

	// put additional states
	for _, v := range testSecondStates {
		_ = manager.PutState(testAccount, &v)
	}
	_ = manager.Update()
	_ = manager.Commit()
	assert.Equal(t, testSecondRoot, manager.GetRoot())

	// get state
	st, _ := manager.GetAccountState(testAccount)
	assert.True(t, stateEquals(&testSecondStates[2], st))

	// set root
	err := manager.SetRoot(testRoot)
	if err != nil {
		t.Errorf("failed to set root: %v", err.Error())
	}
	assert.Equal(t, testRoot, manager.GetRoot())

	// get state after setting root
	st, err = manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get account state: %v", err.Error())
	}
	assert.True(t, stateEquals(&testStates[4], st))
}

func TestStateDBParallel(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	hashFunc := common.HashFuncFactory("sha3")
	manager := NewManager(&store, nil, hashFunc)
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// put states
	assert.Nil(t, manager.GetRoot())
	for _, v := range testStates {
		_ = manager.PutState(testAccount, &v)
	}
	_ = manager.Update()
	_ = manager.Commit()
	assert.Equal(t, testRoot, manager.GetRoot())

	// put additional states
	for _, v := range testSecondStates {
		_ = manager.PutState(testAccount, &v)
	}
	_ = manager.Update()
	_ = manager.Commit()
	assert.Equal(t, testSecondRoot, manager.GetRoot())

	// get state
	st, _ := manager.GetAccountState(testAccount)
	assert.True(t, stateEquals(&testSecondStates[2], st))

	// open another statedb with root hash of previous state
	anotherManager := NewManager(&store, testRoot, hashFunc)
	assert.Equal(t, testRoot, anotherManager.GetRoot())
	assert.Equal(t, testSecondRoot, manager.GetRoot())

	// get state from statedb
	st1, err := manager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get state: %v", err.Error())
	}
	assert.True(t, stateEquals(&testSecondStates[2], st1))

	// get state from another statedb
	st2, err := anotherManager.GetAccountState(testAccount)
	if err != nil {
		t.Errorf("failed to get state: %v", err.Error())
	}
	assert.True(t, stateEquals(&testStates[4], st2))
}
