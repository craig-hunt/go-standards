package signup

import "net/http"

const (
	PathSignups = "/api/signups"
	RouteCreate = http.MethodPost + " " + PathSignups
)

const (
	PlanStarter    Plan = "Starter"
	PlanGrowth     Plan = "Growth"
	PlanEnterprise Plan = "Enterprise"
)

const (
	FieldFullName    = "fullName"
	FieldEmail       = "email"
	FieldPlan        = "plan"
	FieldSeats       = "seats"
	FieldNotes       = "notes"
	FieldAcceptTerms = "acceptTerms"
)

const (
	MsgNameRequired  = "Enter your full name."
	MsgEmailRequired = "Enter your work email."
	MsgEmailInvalid  = "Enter a valid email address."
	MsgPlanRequired  = "Choose a plan."
	MsgPlanUnknown   = "Choose the Starter, Growth, or Enterprise plan."
	MsgSeatsInvalid  = "Enter at least one seat."
	MsgNotesTooLong  = "Keep the notes within the length limit."
	MsgTermsRequired = "Accept the terms to continue."
)

const (
	DefaultSeats   = 1
	MinSeats       = 1
	MaxNotesLength = 1000
	EmailPattern   = `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	summaryFormat  = "%s on the %s plan, %d seat(s)."
)
