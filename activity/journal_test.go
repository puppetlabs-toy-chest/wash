package activity

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHistory(t *testing.T) {
	// Ensure history is empty
	history, jidToHistory = initHistory()

	// Clean up tests at the end.
	defer func() {
		history, jidToHistory = initHistory()
		CloseAll()
	}()

	assert.Equal(t, []Journal{}, History())

	tick := time.Now()
	journal := Journal{ID: "anything", Description: "did something", Start: tick}
	journal.registerCommand()

	assert.Equal(t, []Journal{journal}, History())

	journal.registerCommand()
	assert.Equal(t, []Journal{journal}, History())

	_, err := journal.Open()
	assert.Error(t, err)
}

func TestHistoryWithJournal(t *testing.T) {
	// Ensure history is empty
	history, jidToHistory = initHistory()

	// Clean up tests at the end.
	defer func() {
		history, jidToHistory = initHistory()
		CloseAll()
	}()

	// Log to a journal
	journal := Journal{ID: "anything"}
	Record(context.WithValue(context.Background(), JournalKey, journal), "hello there")
	rdr, err := journal.Open()
	if assert.Nil(t, err) {
		defer rdr.Close()
		bits, err := ioutil.ReadAll(rdr)
		if assert.Nil(t, err) {
			assert.Contains(t, string(bits), "hello there")
		}
	}
}
