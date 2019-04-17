package activity

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPIDToJournalEntry(t *testing.T) {
	journal := JournalForPID(500000)
	assert.Equal(t, "500000", journal.ID)
	assert.Empty(t, journal.Description)

	journal = JournalForPID(os.Getpid())
	expected := strconv.Itoa(os.Getpid()) + "-activity\\.test-[0-9]+"
	assert.Regexp(t, expected, journal.ID)
	assert.Contains(t, journal.Description, "/activity.test -test.testlogfile=")
}
