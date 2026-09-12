package logcapture

import (
	"log/slog"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
)

const (
	firstMessage  = "first event"
	secondMessage = "second event"
	absentMessage = "never logged"
	notJSON       = "plain text\n"
	fieldKey      = "attempt"
	fieldValue    = 2
)

type recorder struct {
	fatals int
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(string, ...any) {}

func (r *recorder) Fatalf(string, ...any) { r.fatals++ }

func TestRecordsReturnsEveryLineInOrder(t *testing.T) {
	logger, capture := New()
	logger.Info(firstMessage, slog.Int(fieldKey, fieldValue))
	logger.Warn(secondMessage)

	records := capture.Records(t)

	expect.Equal(t, len(records), fieldValue)
	expect.Equal(t, records[0][slog.MessageKey], any(firstMessage))
	expect.Equal(t, records[0][fieldKey], any(float64(fieldValue)))
	expect.Equal(t, records[1][slog.MessageKey], any(secondMessage))
	expect.Equal(t, records[1][slog.LevelKey], any(slog.LevelWarn.String()))
}

func TestFindReturnsTheRecordWithTheMessage(t *testing.T) {
	logger, capture := New()
	logger.Info(firstMessage)
	logger.Info(secondMessage)

	expect.Equal(t, capture.Find(t, secondMessage)[slog.MessageKey], any(secondMessage))
}

func TestFindReturnsNilWhenNoRecordCarriesTheMessage(t *testing.T) {
	logger, capture := New()
	logger.Info(firstMessage)

	expect.Equal(t, capture.Find(t, absentMessage), map[string]any(nil))
}

func TestRecordsStopsTheTestOnALineThatIsNotJSON(t *testing.T) {
	_, capture := New()
	_, err := capture.Write([]byte(notJSON))
	expect.NoError(t, err)
	reporter := &recorder{}

	records := capture.Records(reporter)

	expect.Equal(t, reporter.fatals, 1)
	expect.Equal(t, records, []map[string]any(nil))
}

func TestNewCapturesDebugRecords(t *testing.T) {
	logger, capture := New()
	logger.Debug(firstMessage)

	expect.Equal(t, len(capture.Records(t)), 1)
}
