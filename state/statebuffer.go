package state

import (
	"sort"

	"github.com/aergoio/aergo-lib/db"
	"github.com/golang/protobuf/proto"
	"github.com/zhigui-projects/zwasm/types"
)

var (
	emptyBufferEntry = bufferEntry{}
)

type bufferEntry struct {
	key      types.Hash
	dataHash types.Hash
	data     interface{}
}

func newBufferEntry(key types.Hash, hash types.Hash, data interface{}) *bufferEntry {
	return &bufferEntry{
		key:      key,
		dataHash: hash,
		data:     data,
	}
}
func (et *bufferEntry) getKey() []byte {
	return et.key[:]
}
func (et *bufferEntry) getHash() []byte {
	return et.dataHash[:]
}
func (et *bufferEntry) getData() interface{} {
	return et.data
}

type stack []int

func newStack() *stack {
	return &stack{}
}
func (stk *stack) last() int {
	return len(*stk) - 1
}
func (stk *stack) peek() int {
	if stk == nil || len(*stk) == 0 {
		return -1
	}
	return (*stk)[stk.last()]
}
func (stk *stack) pop() int {
	if stk == nil || len(*stk) == 0 {
		return -1
	}
	lst := stk.last()
	top := (*stk)[lst]
	*stk = (*stk)[:lst]
	return top
}
func (stk *stack) push(args ...int) *stack {
	if stk == nil {
		stk = &stack{}
	}
	*stk = append(*stk, args...)
	return stk
}

type bufferIndex map[types.Hash]*stack

func (idxs *bufferIndex) peek(key types.Hash) int {
	return (*idxs)[key].peek()
}
func (idxs *bufferIndex) pop(key types.Hash) int {
	return (*idxs)[key].pop()
}
func (idxs *bufferIndex) push(key types.Hash, argv ...int) {
	(*idxs)[key] = (*idxs)[key].push(argv...)
}
func (idxs *bufferIndex) rollback(snapshot int) {
	for k, v := range *idxs {
		for v.peek() >= snapshot {
			v.pop()
		}
		if v.peek() < 0 {
			delete(*idxs, k)
		}
	}
}

type stateBuffer struct {
	entries []bufferEntry
	indexes bufferIndex
	nextIdx int
	hasher  func(data ...[]byte) []byte
}

func newStateBuffer(hasher func(data ...[]byte) []byte) *stateBuffer {
	buffer := stateBuffer{
		entries: []bufferEntry{},
		indexes: bufferIndex{},
		nextIdx: 0,
		hasher:  hasher,
	}
	return &buffer
}

func (buffer *stateBuffer) reset() error {
	return buffer.rollback(0)
}

func (buffer *stateBuffer) get(key types.Hash) *bufferEntry {
	if index, ok := buffer.indexes[key]; ok {
		return &buffer.entries[index.peek()]
	}
	return nil
}

func (buffer *stateBuffer) put(key types.Hash, data interface{}) error {
	snapshot := buffer.snapshot()
	hash, err := getHash(data, buffer.hasher)
	if err != nil {
		return err
	}
	et := newBufferEntry(key, hash, data)
	buffer.entries = append(buffer.entries, *et)
	buffer.indexes[key] = buffer.indexes[key].push(snapshot)
	buffer.nextIdx++
	return nil
}

func (buffer *stateBuffer) snapshot() int {
	return buffer.nextIdx
}

func (buffer *stateBuffer) rollback(snapshot int) error {
	for i := buffer.nextIdx - 1; i >= snapshot; i-- {
		et := buffer.entries[i]
		buffer.indexes.pop(et.key)
		if buffer.indexes.peek(et.key) < 0 {
			delete(buffer.indexes, et.key)
		}
	}
	buffer.entries = buffer.entries[:snapshot]
	//buffer.indexes.rollback(snapshot)
	buffer.nextIdx = snapshot
	return nil
}

func (buffer *stateBuffer) isEmpty() bool {
	return len(buffer.entries) == 0
}

func (buffer *stateBuffer) export() ([][]byte, [][]byte) {
	size := len(buffer.indexes)
	bufs := make([]bufferEntry, 0, size)
	for _, v := range buffer.indexes {
		idx := v.peek()
		if idx < 0 {
			continue
		}
		bufs = append(bufs, buffer.entries[idx])
	}
	sort.Slice(bufs, func(i, j int) bool {
		return -1 == (bufs[i].key).Compare(bufs[j].key)
	})
	keys := make([][]byte, size)
	vals := make([][]byte, size)
	for i, et := range bufs {
		keys[i] = append(keys[i], et.getKey()...)
		vals[i] = append(vals[i], et.getHash()...)
	}
	return keys, vals
}

func (buffer *stateBuffer) commit(store *db.DB) error {
	dbtx := (*store).NewTx()
	for _, v := range buffer.indexes {
		et := buffer.entries[v.peek()]
		buf, err := marshal(et.data)
		if err != nil {
			dbtx.Discard()
			return err
		}
		dbtx.Set(et.getHash(), buf)
	}
	dbtx.Commit()
	return nil
}

func marshal(data interface{}) ([]byte, error) {
	switch data.(type) {
	case []byte:
		return data.([]byte), nil
	case *[]byte:
		return *(data.(*[]byte)), nil
	case proto.Message:
		return proto.Marshal(data.(proto.Message))
	}
	return nil, nil
}

func getHash(data interface{}, hasher func(data ...[]byte) []byte) (types.Hash, error) {
	switch data.(type) {
	case types.ImplHash:
		return data.(types.ImplHash).Hash(), nil
	default:
	}
	buf, err := marshal(data)
	if err != nil {
		return types.EmptyHash, err
	}
	return types.GetHash(buf, hasher), nil
}
