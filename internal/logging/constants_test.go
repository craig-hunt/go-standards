package logging

import "time"

const (
	testService      = "billing"
	testMessage      = "invoice issued"
	attachedTimeKey  = "deadline"
	attachedGroup    = "schedule"
	zoneName         = "EDT"
	zoneOffset       = -4 * 60 * 60
	suppressedReason = "debug records stay suppressed at info level"
)

var sampleInstant = time.Date(2026, time.September, 12, 9, 30, 0, 0, time.FixedZone(zoneName, zoneOffset))
