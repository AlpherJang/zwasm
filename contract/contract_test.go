package contract

import (
	"testing"
	"github.com/zhigui-projects/zwasm/types"
	"github.com/gogo/protobuf/proto"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	sCode, _ := loadCode()
	sCodeLen := len(sCode)

	ci := &types.CallInfo{Name: "invoke", Args: [][]byte{[]byte("abc"), []byte("xyz")}}
	ciBuf, _ := proto.Marshal(ci)

	sCodeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sCodeBytes, uint32(sCodeLen))
	sCodeBytes = append(sCodeBytes, sCode...)

	sTotalBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sTotalBytes, uint32(4+4+sCodeLen))
	sTotalBytes = append(sTotalBytes, sCodeBytes...)
	sTotalBytes = append(sTotalBytes, ciBuf...)

	store := createDB(t)
	defer closeDB(t, store)
	crtState, _ := createContractState(t, store)

	context := &Context{gasLimit: 10000, senderAddress: []byte("sender")}
	_, gas, _ := Create(crtState, context, sTotalBytes)
	assert.True(t, gas > uint64(sCodeLen/1024*gasByKBSize))
	creator, _ := crtState.GetData([]byte("Creator"))
	assert.Equal(t, context.senderAddress, creator)
}
