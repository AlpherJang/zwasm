package state

import (
	"bytes"
	"github.com/aergoio/aergo-lib/db"

	"github.com/aergoio/aergo/pkg/trie"
	"github.com/zhigui-projects/zwasm/types"
)

func (mgr *Manager) OpenContractStateAccount(aid types.AccountID) (*ContractState, error) {
	st, err := mgr.GetAccountState(aid)
	if err != nil {
		return nil, err
	}
	return mgr.OpenContractState(st)
}

func (mgr *Manager) OpenContractState(crtState *types.State) (*ContractState, error) {
	res := &ContractState{
		State:   crtState,
		storage: trie.NewTrie(nil, mgr.hasher, *mgr.store),
		buffer:  newStateBuffer(mgr.hasher),
		store:   mgr.store,
		hasher:  mgr.hasher,
	}
	if crtState.StorageRoot != nil && !types.EmptyHash.Equal(types.ToHash(crtState.StorageRoot)) {
		res.storage.Root = crtState.StorageRoot
	}
	return res, nil
}

func (mgr *Manager) CommitContractState(crtState *ContractState) error {
	defer func() {
		if bytes.Compare(crtState.State.StorageRoot, crtState.storage.Root) != 0 {
			crtState.State.StorageRoot = crtState.storage.Root
		}
		crtState.storage = nil
	}()

	if crtState.buffer.isEmpty() {
		// do nothing
		return nil
	}

	keys, vals := crtState.buffer.export()
	_, err := crtState.storage.Update(keys, vals)
	if err != nil {
		return err
	}
	crtState.buffer.commit(crtState.store)

	err = crtState.storage.Commit()
	if err != nil {
		return err
	}
	return crtState.buffer.reset()
}

type ContractState struct {
	*types.State
	code    []byte
	storage *trie.Trie
	buffer  *stateBuffer
	store   *db.DB
	hasher  func(data ...[]byte) []byte
}

func (crtState *ContractState) SetNonce(nonce uint64) {
	crtState.State.Nonce = nonce
}
func (crtState *ContractState) GetNonce() uint64 {
	return crtState.State.GetNonce()
}

func (crtState *ContractState) SetBalance(balance uint64) {
	crtState.State.Balance = balance
}
func (crtState *ContractState) GetBalance() uint64 {
	return crtState.State.GetBalance()
}

func (crtState *ContractState) SetCode(code []byte) error {
	codeHash := crtState.hasher(code)
	err := saveData(crtState.store, codeHash[:], &code)
	if err != nil {
		return err
	}
	crtState.State.CodeHash = codeHash[:]
	return nil
}
func (crtState *ContractState) GetCode() ([]byte, error) {
	if crtState.code != nil {
		// already loaded.
		return crtState.code, nil
	}
	codeHash := crtState.State.GetCodeHash()
	if codeHash == nil {
		// not defined. do nothing.
		return nil, nil
	}
	err := loadData(crtState.store, crtState.State.CodeHash, &crtState.code)
	if err != nil {
		return nil, err
	}
	return crtState.code, nil
}

func (crtState *ContractState) SetData(key, value []byte) error {
	return crtState.buffer.put(types.GetHash(key, crtState.hasher), value)
}

func (crtState *ContractState) GetData(key []byte) ([]byte, error) {
	id := types.GetHash(key, crtState.hasher)
	entry := crtState.buffer.get(id)
	if entry != nil {
		return entry.data.([]byte), nil
	}
	dkey, err := crtState.storage.Get(id[:])
	if err != nil {
		return nil, err
	}
	if len(dkey) == 0 {
		return nil, nil
	}
	value := []byte{}
	err = loadData(crtState.store, dkey, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}
