// Package signup validates and records account signups. Validation reports
// every field problem at once, so a form marks all of them in one round trip.
package signup

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Plan string

type ID int64

// Request mirrors the form. Seats is a pointer so an omitted count takes the
// default while an explicit zero still fails validation.
type Request struct {
	FullName    string `json:"fullName"`
	Email       string `json:"email"`
	Plan        Plan   `json:"plan"`
	Seats       *int   `json:"seats"`
	Notes       string `json:"notes"`
	AcceptTerms bool   `json:"acceptTerms"`
}

type Signup struct {
	FullName string
	Email    string
	Plan     Plan
	Seats    int
	Notes    string
}

type Confirmation struct {
	ID      ID     `json:"id"`
	Summary string `json:"summary"`
}

type Store interface {
	Save(ctx context.Context, signup Signup) (ID, error)
}

var emailPattern = regexp.MustCompile(EmailPattern)

func (p Plan) Known() bool {
	switch p {
	case PlanStarter, PlanGrowth, PlanEnterprise:
		return true
	default:
		return false
	}
}

func Validate(request Request) (Signup, map[string]string) {
	problems := map[string]string{}

	name := strings.TrimSpace(request.FullName)
	if name == "" {
		problems[FieldFullName] = MsgNameRequired
	}

	email := strings.TrimSpace(request.Email)
	switch {
	case email == "":
		problems[FieldEmail] = MsgEmailRequired
	case !emailPattern.MatchString(email):
		problems[FieldEmail] = MsgEmailInvalid
	}

	switch {
	case request.Plan == "":
		problems[FieldPlan] = MsgPlanRequired
	case !request.Plan.Known():
		problems[FieldPlan] = MsgPlanUnknown
	}

	seats := DefaultSeats
	if request.Seats != nil {
		seats = *request.Seats
		if seats < MinSeats {
			problems[FieldSeats] = MsgSeatsInvalid
		}
	}

	notes := strings.TrimSpace(request.Notes)
	if utf8.RuneCountInString(notes) > MaxNotesLength {
		problems[FieldNotes] = MsgNotesTooLong
	}

	if !request.AcceptTerms {
		problems[FieldAcceptTerms] = MsgTermsRequired
	}

	if len(problems) > 0 {
		return Signup{}, problems
	}
	return Signup{FullName: name, Email: email, Plan: request.Plan, Seats: seats, Notes: notes}, nil
}

func (s Signup) Summary() string {
	return fmt.Sprintf(summaryFormat, s.FullName, s.Plan, s.Seats)
}
