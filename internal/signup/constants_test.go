package signup_test

import "github.com/craig-hunt/go-standards/internal/signup"

const (
	sampleName      = "Dana Whitfield"
	paddedName      = "  Dana Whitfield  "
	sampleEmail     = "dana.whitfield@example.com"
	paddedEmail     = " dana.whitfield@example.com "
	invalidEmail    = "dana.whitfield.example.com"
	whitespace      = "   "
	unknownPlan     = signup.Plan("Platinum")
	sampleSeats     = 12
	sampleNotes     = "Migrating from a competitor next quarter."
	paddedNotes     = " Migrating from a competitor next quarter. "
	filler          = "n"
	storeFailure    = "database offline"
	savedID         = signup.ID(7)
	unknownField    = "referrer"
	expectedSummary = "Dana Whitfield on the Growth plan, 12 seat(s)."
)
