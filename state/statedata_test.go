package state

import (
	"testing"

	"fmt"
	"os"

	"github.com/aergoio/aergo-lib/db"
	"github.com/stretchr/testify/assert"
)

var (
	testKey  = []byte("test_key")
	testData = []byte("test_data")
	testOver = []byte("test_over")
)

func TestStateDataBasic(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// save data
	if err := saveData(&store, testKey, &testData); err != nil {
		t.Errorf("failed to save data: %v", err.Error())
	}

	// load data
	data := []byte{}
	if err := loadData(&store, testKey, &data); err != nil {
		t.Errorf("failed to load data: %v", err.Error())
	}
	assert.NotNil(t, data)
	assert.Equal(t, testData, data)
}

func TestStateDataNil(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// load data before saving
	var data interface{}
	assert.Nil(t, data)
	if err := loadData(&store, testKey, &data); err != nil {
		t.Errorf("failed to load data: %v", err.Error())
	}
	assert.Nil(t, data)
}

func TestStateDataEmpty(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// save empty data
	var testEmpty []byte
	if err := saveData(&store, testKey, &testEmpty); err != nil {
		t.Errorf("failed to save nil data: %v", err.Error())
	}

	// load empty data
	data := []byte{}
	if err := loadData(&store, testKey, &data); err != nil {
		t.Errorf("failed to load data: %v", err.Error())
	}
	fmt.Println(len(data))
	assert.NotNil(t, data)
	assert.Empty(t, data)
}

func TestStateDataOverwrite(t *testing.T) {
	store := db.NewDB(db.BadgerImpl, t.Name())
	defer func() {
		store.Close()
		os.RemoveAll(t.Name())
	}()

	// save data
	if err := saveData(&store, testKey, &testData); err != nil {
		t.Errorf("failed to save data: %v", err.Error())
	}

	// save another data to same key
	if err := saveData(&store, testKey, &testOver); err != nil {
		t.Errorf("failed to overwrite data: %v", err.Error())
	}

	// load data
	data := []byte{}
	if err := loadData(&store, testKey, &data); err != nil {
		t.Errorf("failed to load data: %v", err.Error())
	}
	assert.NotNil(t, data)
	assert.Equal(t, testOver, data)
}
