package types

import (
	"bytes"
	"reflect"
	"encoding/hex"
)

// Hash is a fixed size bytes
type Hash [32]byte

// BlockID is a Hash to identify a block
type BlockID Hash

// AccountID is a Hash to identify an account
type AccountID Hash

// TxID is a Hash to identify a transaction
type TxID Hash

type StateProof struct {
	State     *State
	Inclusion bool
	ProofKey  []byte
	ProofVal  []byte
	AuditPath [][]byte
}

// ImplHash is a object has Hash
type ImplHash interface {
	Hash() Hash
}

var (
	EmptyHash      = Hash{}
	EmptyAccountID = AccountID{}
)

// GetHash make a Hash from hash of bytes
func GetHash(bytes []byte, hashFunc func(data ...[]byte) []byte) Hash {
	hash := hashFunc(bytes)
	return ToHash(hash)
}

// ToHash make a Hash from bytes
func ToHash(hash []byte) Hash {
	buf := Hash{}
	copy(buf[:], hash)
	return Hash(buf)
}
func (id Hash) String() string {
	return hex.EncodeToString(id[:])
}

// Bytes make a byte slice from id
func (id Hash) Bytes() []byte {
	if id == EmptyHash {
		return nil
	}
	return id[:]
}

// Compare returns an integer comparing two Hashs as byte slices.
func (id Hash) Compare(alt Hash) int {
	return bytes.Compare(id.Bytes(), alt.Bytes())
}

// Equal returns a boolean comparing two Hashs as byte slices.
func (id Hash) Equal(alt Hash) bool {
	return bytes.Equal(id.Bytes(), alt.Bytes())
}

// ToBlockID make a BlockID from bytes
func ToBlockID(blockHash []byte) BlockID {
	return BlockID(ToHash(blockHash))
}
func (id BlockID) String() string {
	return Hash(id).String()
}

// ToTxID make a TxID from bytes
func ToTxID(txHash []byte) TxID {
	return TxID(ToHash(txHash))
}
func (id TxID) String() string {
	return Hash(id).String()
}

// ToAccountID make a AccountHash from bytes
func ToAccountID(account []byte, hashFunc func(data ...[]byte) []byte) AccountID {
	return AccountID(GetHash(account, hashFunc))
}
func (id AccountID) String() string {
	return Hash(id).String()
}

// NewState returns an instance of account state
func NewState() *State {
	return &State{
		Nonce:   0,
		Balance: 0,
	}
}

// func (st *State) IsEmpty() bool {
// 	return st.Nonce == 0 && st.Balance == 0
// }

// func (st *State) GetHash() []byte {
// 	digest := sha256.New()
// 	binary.Write(digest, binary.LittleEndian, st.Nonce)
// 	binary.Write(digest, binary.LittleEndian, st.Balance)
// 	return digest.Sum(nil)
// }

// func (st *State) Clone() *State {
// 	if st == nil {
// 		return nil
// 	}
// 	return &State{
// 		Nonce:       st.Nonce,
// 		Balance:     st.Balance,
// 		CodeHash:    st.CodeHash,
// 		StorageRoot: st.StorageRoot,
// 	}
// }

func Clone(i interface{}) interface{} {
	if i == nil {
		return nil
	}
	return reflect.Indirect(reflect.ValueOf(i)).Interface()
}
