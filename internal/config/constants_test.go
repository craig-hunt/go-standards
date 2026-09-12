package config

import "time"

const (
	testDatabaseURL     = "postgres://demo@db.example.test/demo"
	testAPIToken        = "test-token"
	testServiceName     = "inventory-api"
	testPort            = "9090"
	testPortValue       = 9090
	testLogLevel        = "debug"
	testTimeout         = "3s"
	testTimeoutValue    = 3 * time.Second
	portNotANumber      = "eighty"
	unknownLogLevel     = "loud"
	unparsableDuration  = "soon"
	zeroDuration        = "0s"
	negativeDuration    = "-1s"
	missingKeyMessage   = "the error names the missing variable"
	invalidKeyMessage   = "the error names the invalid variable"
	smallestPositiveDur = "1ns"
)
