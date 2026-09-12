// Package logcapture records what a logger writes, so tests assert on the
// structured fields a request produced rather than on raw text.
package logcapture

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/logging"
)

const (
	Service           = "capture"
	formatUnparseable = "log line %q is not JSON: %v"
)

// Capture guards its buffer because servers under test log from their own
// goroutines while the test reads.
type Capture struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func New() (*slog.Logger, *Capture) {
	capture := &Capture{}
	return logging.New(capture, Service, slog.LevelDebug), capture
}

func (c *Capture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buffer.Write(p)
}

func (c *Capture) Records(r expect.Reporter) []map[string]any {
	r.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	var records []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(c.buffer.Bytes()))
	for scanner.Scan() {
		var fields map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &fields); err != nil {
			r.Fatalf(formatUnparseable, scanner.Text(), err)
			return nil
		}
		records = append(records, fields)
	}
	return records
}

// Find returns the first record carrying the message, or nil when none does.
func (c *Capture) Find(r expect.Reporter, message string) map[string]any {
	r.Helper()
	for _, record := range c.Records(r) {
		if record[slog.MessageKey] == message {
			return record
		}
	}
	return nil
}
