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
	history = initHistory()

	// Clean up tests at the end.
	defer func() {
		history = initHistory()
		CloseAll()
	}()

	assert.Equal(t, []Journal{}, History())

	tick := time.Now()
	journal := Journal{ID: "anything", Description: "did something", start: tick}
	journal.addToHistory()

	assert.Equal(t, []Journal{journal}, History())

	journal.addToHistory()
	assert.Equal(t, []Journal{journal}, History())

	_, err := journal.Open()
	assert.Error(t, err)
}

func TestHistoryWithJournal(t *testing.T) {
	// Ensure history is empty
	history = initHistory()

	// Clean up tests at the end.
	defer func() {
		history = initHistory()
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
