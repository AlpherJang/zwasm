package contract

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadCode() ([]byte, error) {
	code, err := ioutil.ReadFile("fixture/contract.wasm")
	if err != nil {
		return nil, err
	}

	return code, nil
}

func TestCreate(t *testing.T) {
	code, err := loadCode()
	assert.NoError(t, err)
	assert.NotNil(t, code)
}
