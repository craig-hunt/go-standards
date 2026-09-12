package expect

const (
	sampleValue      = 7
	otherValue       = 8
	sampleMessage    = "sample failure"
	sampleCondition  = "the condition holds"
	countMismatch    = "%s: got %d calls, want %d"
	formatMismatchIn = "got format %q, want %q"
	errorsCalls      = "Errorf"
	fatalCalls       = "Fatalf"
	helperCalls      = "Helper"
	wrapFormat       = "%w"
)
