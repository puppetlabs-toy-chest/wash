package activity

import (
	"context"
	"io/ioutil"
	"regexp"
	"sync"
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
	ctx := context.WithValue(context.Background(), JournalKey, journal)
	Record(ctx, "hello there")
	Warnf(ctx, "not good")
	rdr, err := journal.Open()
	if assert.Nil(t, err) {
		defer rdr.Close()
		bits, err := ioutil.ReadAll(rdr)
		if assert.Nil(t, err) {
			assert.Regexp(t, regexp.MustCompile("info.*hello there"), string(bits))
			assert.Regexp(t, regexp.MustCompile("warn.*not good"), string(bits))
		}
	}
}

func TestRecorder_CanRecordMethodInvocations(t *testing.T) {
	recorder := newRecorder()
	var invoked bool
	invoker := func() {
		invoked = true
	}

	assert.False(t, recorder.methodInvoked("foo", "List"))
	recorder.submitMethodInvocation("foo", "List", invoker)
	assert.True(t, recorder.methodInvoked("foo", "List"))
	assert.True(t, invoked)

	invoked = false
	recorder.submitMethodInvocation("foo", "List", invoker)
	assert.False(t, invoked)

	// Test a different method
	invoked = false
	assert.False(t, recorder.methodInvoked("foo", "Exec"))
	recorder.submitMethodInvocation("foo", "Exec", invoker)
	assert.True(t, recorder.methodInvoked("foo", "Exec"))
	assert.True(t, invoked)
}

func TestRecorder_RecordsMethodInvocationsOnce(t *testing.T) {
	recorder := newRecorder()
	var count int
	invoker := func() { count++ }
	var wg sync.WaitGroup
	for i := 0; i <= 100; i++ {
		wg.Add(1)
		go func() {
			recorder.submitMethodInvocation("foo", "Read", invoker)
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, count)
	assert.True(t, recorder.methodInvoked("foo", "Read"))
}
