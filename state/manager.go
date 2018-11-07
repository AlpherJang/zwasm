package state

import (
	"sync"
	"github.com/aergoio/aergo/pkg/trie"
	"errors"
	"github.com/zhigui-projects/zwasm/types"
	"fmt"
	"github.com/aergoio/aergo-lib/db"
)

var (
	errSaveData      = errors.New("failed to save data: invalid key")
	errLoadData      = errors.New("failed to load data: invalid key")
	errSaveStateData = errors.New("failed to save StateData: invalid HashID")
	errLoadStateData = errors.New("failed to load StateData: invalid HashID")
)

var (
	errInvalidArgs = errors.New("invalid arguments")
	errInvalidRoot = errors.New("invalid root")
	errSetRoot     = errors.New("failed to set root: invalid root")
	errLoadRoot    = errors.New("failed to load root: invalid root")
	errGetState    = errors.New("failed to get state: invalid account id")
	errPutState    = errors.New("failed to put state: invalid account id")
)

type Manager struct {
	lock   sync.RWMutex
	trie   *trie.Trie
	buffer *stateBuffer
	store  *db.DB
	hasher func(data ...[]byte) []byte
}

func NewManager(store *db.DB, root []byte, hasher func(data ...[]byte) []byte) *Manager {
	manager := &Manager{
		trie:   trie.NewTrie(root, hasher, *store),
		buffer: newStateBuffer(hasher),
		store:  store,
		hasher: hasher,
	}

	return manager
}

// Clone returns a new state Manager which has same store and Root
func (mgr *Manager) Clone() *Manager {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	return NewManager(mgr.store, mgr.GetRoot(), mgr.hasher)
}

// GetRoot returns root dataHash of trie
func (mgr *Manager) GetRoot() []byte {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	return mgr.trie.Root
}

// SetRoot updates root node of trie as a given root dataHash
func (mgr *Manager) SetRoot(root []byte) error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	// update root node
	mgr.trie.Root = root
	// reset buffer
	return mgr.buffer.reset()
}

// LoadCache reads first layer of trie given root dataHash
// and also updates root node of trie as a given root dataHash
func (mgr *Manager) LoadCache(root []byte) error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	// update root node and load cache
	err := mgr.trie.LoadCache(root)
	if err != nil {
		return err
	}
	// reset buffer
	return mgr.buffer.reset()
}

// Revert rollbacks trie to previous root dataHash
func (mgr *Manager) Revert(root types.Hash) error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	// // handle nil bytes
	// targetRoot := root.Bytes()

	// // revert trie
	// err := mgr.trie.Revert(targetRoot)
	// if err != nil {
	// 	// when targetRoot is not contained in the cached tries.
	// 	mgr.trie.Root = targetRoot
	// }

	// just update root node as targetRoot.
	// revert trie consumes unnecessarily long time.
	mgr.trie.Root = root.Bytes()

	// reset buffer
	return mgr.buffer.reset()
}

// PutState puts account id and its state into state buffer.
func (mgr *Manager) PutState(id types.AccountID, state *types.State) error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	if id == types.EmptyAccountID {
		return errPutState
	}
	return mgr.buffer.put(types.Hash(id), state)
}

// GetAccountState gets state of account id from state manager.
// empty state is returned when there is no state corresponding to account id.
func (mgr *Manager) GetAccountState(aid types.AccountID) (*types.State, error) {
	st, err := mgr.GetState(aid)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return &types.State{}, nil
	}
	return st, nil
}

type RolledState struct {
	mgr    *Manager
	id     []byte
	aid    types.AccountID
	oldV   *types.State
	newV   *types.State
	newOne bool
	create bool
}

func (v *RolledState) ID() []byte {
	return v.id
}

func (v *RolledState) AccountID() types.AccountID {
	return v.aid
}

func (v *RolledState) State() *types.State {
	return v.newV
}

func (v *RolledState) SetNonce(nonce uint64) {
	v.newV.Nonce = nonce
}

func (v *RolledState) Balance() uint64 {
	return v.newV.Balance
}

func (v *RolledState) AddBalance(amount uint64) {
	v.newV.Balance += amount
}

func (v *RolledState) SubBalance(amount uint64) {
	v.newV.Balance -= amount
}

func (v *RolledState) IsNew() bool {
	return v.newOne
}

func (v *RolledState) IsCreate() bool {
	return v.create
}

func (v *RolledState) Reset() {
	*v.newV = types.State(*v.oldV)
}

func (v *RolledState) PutState() error {
	return v.mgr.PutState(v.aid, v.newV)
}

func (mgr *Manager) CreateRolledAccountState(id []byte) (*RolledState, error) {
	v, err := mgr.GetRolledAccountState(id)
	if err != nil {
		return nil, err
	}
	if !v.newOne {
		return nil, fmt.Errorf("account(%x) aleardy exists", v.ID())
	}
	v.create = true
	return v, nil
}

func (mgr *Manager) GetRolledAccountState(id []byte) (*RolledState, error) {
	aid := types.ToAccountID(id, mgr.hasher)
	st, err := mgr.GetState(aid)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return &RolledState{
			mgr:    mgr,
			id:     id,
			aid:    aid,
			oldV:   &types.State{},
			newV:   &types.State{},
			newOne: true,
		}, nil
	}
	newV := new(types.State)
	*newV = types.State(*st)
	return &RolledState{
		mgr:  mgr,
		id:   id,
		aid:  aid,
		oldV: st,
		newV: newV,
	}, nil
}

// GetState gets state of account id from state buffer and trie.
// nil value is returned when there is no state corresponding to account id.
func (mgr *Manager) GetState(id types.AccountID) (*types.State, error) {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	if id == types.EmptyAccountID {
		return nil, errGetState
	}
	// get state from buffer
	entry := mgr.buffer.get(types.Hash(id))
	if entry != nil {
		return entry.getData().(*types.State), nil
	}
	// get state from trie
	return mgr.getState(id)
}

// getState gets state of account id from trie.
// nil value is returned when there is no state corresponding to account id.
func (mgr *Manager) getState(id types.AccountID) (*types.State, error) {
	key, err := mgr.trie.Get(id[:])
	if err != nil {
		return nil, err
	}
	if key == nil || len(key) == 0 {
		return nil, nil
	}
	return mgr.loadStateData(key)
}

// GetStateAndProof gets the state and associated proof of an account
// in the given trie root. If the account doesnt exist, a proof of
// non existence is returned.
func (mgr *Manager) GetStateAndProof(id types.AccountID, root []byte) (*types.StateProof, error) {
	var state *types.State
	var ap [][]byte
	var proofKey, proofVal []byte
	var isIncluded bool
	var err error
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	if len(root) != 0 {
		// Get the state and proof of the account for a past state
		ap, isIncluded, proofKey, proofVal, err = mgr.trie.MerkleProofPast(id[:], root)
		if err != nil {
			return nil, err
		}
	} else {
		// Get the state and proof of the account
		// The wallet should check that state hashes to proofVal and verify the audit path,
		// The returned proofVal shouldn't be trusted by the wallet, it is used to proove non inclusion
		ap, isIncluded, proofKey, proofVal, err = mgr.trie.MerkleProof(id[:])
		if err != nil {
			return nil, err
		}
	}
	if isIncluded {
		state, err = mgr.loadStateData(proofVal)
		if err != nil {
			return nil, err
		}
	}
	stateProof := &types.StateProof{
		State:     state,
		Inclusion: isIncluded,
		ProofKey:  proofKey,
		ProofVal:  proofVal,
		AuditPath: ap,
	}
	return stateProof, nil
}

// Snapshot represents revision number of statedb
type Snapshot int

// Snapshot returns revision number of state buffer
func (mgr *Manager) Snapshot() Snapshot {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	return Snapshot(mgr.buffer.snapshot())
}

// Rollback discards changes of state buffer to revision number
func (mgr *Manager) Rollback(revision Snapshot) error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	return mgr.buffer.rollback(int(revision))
}

// Update applies changes of state buffer to trie
func (mgr *Manager) Update() error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	keys, vals := mgr.buffer.export()
	if len(keys) == 0 || len(vals) == 0 {
		// nothing to update
		return nil
	}
	_, err := mgr.trie.Update(keys, vals)
	if err != nil {
		return err
	}
	return nil
}

// Commit writes state buffer and trie to db
func (mgr *Manager) Commit() error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	err := mgr.trie.Commit()
	if err != nil {
		return err
	}
	err = mgr.buffer.commit(mgr.store)
	if err != nil {
		return err
	}
	return mgr.buffer.reset()
}
