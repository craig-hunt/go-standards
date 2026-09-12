package task_test

import "github.com/craig-hunt/go-standards/internal/task"

const (
	sampleTitle      = "Draft the incident postmortem"
	paddedTitle      = "  Draft the incident postmortem  "
	whitespaceTitle  = "   "
	filler           = "a"
	multibyteFiller  = "é"
	unknownField     = "priority"
	unknownFilter    = "archived"
	notANumber       = "seven"
	zeroID           = "0"
	negativeID       = "-3"
	validID          = "42"
	validIDValue     = task.ID(42)
	missingID        = task.ID(404)
	storeFailure     = "database offline"
	pathSeparator    = "/"
	querySeparator   = "?"
	firstSeededID    = task.ID(1)
	secondSeededID   = task.ID(2)
	createdID        = task.ID(3)
	oneRemoved       = int64(1)
	remainingAfter   = 1
	storeLengthAfter = 1
)
