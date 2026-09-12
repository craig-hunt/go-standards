package signup_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/craig-hunt/go-standards/internal/apitest"
	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/logcapture"
	"github.com/craig-hunt/go-standards/internal/signup"
)

func validRequest() signup.Request {
	seats := sampleSeats
	return signup.Request{
		FullName:    sampleName,
		Email:       sampleEmail,
		Plan:        signup.PlanGrowth,
		Seats:       &seats,
		Notes:       sampleNotes,
		AcceptTerms: true,
	}
}

func validSignup() signup.Signup {
	return signup.Signup{FullName: sampleName, Email: sampleEmail, Plan: signup.PlanGrowth, Seats: sampleSeats, Notes: sampleNotes}
}

func seatsOf(count int) *int {
	return &count
}

type recordingStore struct {
	saved []signup.Signup
	err   error
}

func (s *recordingStore) Save(_ context.Context, record signup.Signup) (signup.ID, error) {
	if s.err != nil {
		return 0, s.err
	}
	s.saved = append(s.saved, record)
	return savedID, nil
}

func post(t *testing.T, store signup.Store, body any) (int, []byte) {
	t.Helper()
	logger, _ := logcapture.New()
	mux := http.NewServeMux()
	signup.NewHandler(store, logger).Register(mux)
	recorder := apitest.Do(t, mux, apitest.Request{Method: http.MethodPost, Target: signup.PathSignups, Body: body})
	return recorder.Code, recorder.Body.Bytes()
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var value T
	expect.NoError(t, json.Unmarshal(raw, &value))
	return value
}

func TestValidateAcceptsACompleteRequest(t *testing.T) {
	record, problems := signup.Validate(validRequest())

	expect.Equal(t, problems, map[string]string(nil))
	expect.Equal(t, record, validSignup())
}

func TestValidateTrimsTheTextFields(t *testing.T) {
	request := validRequest()
	request.FullName, request.Email, request.Notes = paddedName, paddedEmail, paddedNotes

	record, problems := signup.Validate(request)

	expect.Equal(t, problems, map[string]string(nil))
	expect.Equal(t, record, validSignup())
}

func TestValidateUsesTheDefaultSeatCountWhenNoneIsGiven(t *testing.T) {
	request := validRequest()
	request.Seats = nil

	record, _ := signup.Validate(request)

	expect.Equal(t, record.Seats, signup.DefaultSeats)
}

func TestValidateAcceptsTheBoundaryValues(t *testing.T) {
	request := validRequest()
	request.Seats = seatsOf(signup.MinSeats)
	request.Notes = strings.Repeat(filler, signup.MaxNotesLength)

	record, problems := signup.Validate(request)

	expect.Equal(t, problems, map[string]string(nil))
	expect.Equal(t, record.Seats, signup.MinSeats)
}

func TestValidateNamesEveryMissingFieldAtOnce(t *testing.T) {
	_, problems := signup.Validate(signup.Request{})

	expect.Equal(t, problems, map[string]string{
		signup.FieldFullName:    signup.MsgNameRequired,
		signup.FieldEmail:       signup.MsgEmailRequired,
		signup.FieldPlan:        signup.MsgPlanRequired,
		signup.FieldAcceptTerms: signup.MsgTermsRequired,
	})
}

func TestValidateRejectsEachInvalidValue(t *testing.T) {
	cases := []struct {
		name   string
		change func(*signup.Request)
		field  string
		want   string
	}{
		{name: "blank name", change: func(r *signup.Request) { r.FullName = whitespace }, field: signup.FieldFullName, want: signup.MsgNameRequired},
		{name: "malformed email", change: func(r *signup.Request) { r.Email = invalidEmail }, field: signup.FieldEmail, want: signup.MsgEmailInvalid},
		{name: "unknown plan", change: func(r *signup.Request) { r.Plan = unknownPlan }, field: signup.FieldPlan, want: signup.MsgPlanUnknown},
		{name: "seats below the minimum", change: func(r *signup.Request) { r.Seats = seatsOf(signup.MinSeats - 1) }, field: signup.FieldSeats, want: signup.MsgSeatsInvalid},
		{name: "notes over the limit", change: func(r *signup.Request) { r.Notes = strings.Repeat(filler, signup.MaxNotesLength+1) }, field: signup.FieldNotes, want: signup.MsgNotesTooLong},
		{name: "terms not accepted", change: func(r *signup.Request) { r.AcceptTerms = false }, field: signup.FieldAcceptTerms, want: signup.MsgTermsRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := validRequest()
			tc.change(&request)

			record, problems := signup.Validate(request)

			expect.Equal(t, problems, map[string]string{tc.field: tc.want})
			expect.Equal(t, record, signup.Signup{})
		})
	}
}

func TestPlanKnownRecognizesOnlyTheOfferedPlans(t *testing.T) {
	for _, plan := range []signup.Plan{signup.PlanStarter, signup.PlanGrowth, signup.PlanEnterprise} {
		expect.Equal(t, plan.Known(), true)
	}
	expect.Equal(t, unknownPlan.Known(), false)
}

func TestSummaryDescribesTheSignup(t *testing.T) {
	expect.Equal(t, validSignup().Summary(), expectedSummary)
}

func TestRequestJSONNamesMatchTheFieldConstants(t *testing.T) {
	encoded, err := json.Marshal(validRequest())
	expect.NoError(t, err)

	fields := decode[map[string]any](t, encoded)

	expect.Equal(t, slices.Sorted(maps.Keys(fields)), slices.Sorted(slices.Values([]string{
		signup.FieldFullName, signup.FieldEmail, signup.FieldPlan, signup.FieldSeats, signup.FieldNotes, signup.FieldAcceptTerms,
	})))
}

func TestCreateSavesTheSignupAndConfirmsIt(t *testing.T) {
	store := &recordingStore{}

	status, body := post(t, store, validRequest())

	expect.Equal(t, status, http.StatusCreated)
	expect.Equal(t, decode[signup.Confirmation](t, body), signup.Confirmation{ID: savedID, Summary: expectedSummary})
	expect.Equal(t, store.saved, []signup.Signup{validSignup()})
}

func TestCreateReturnsEveryFieldProblemWithoutSaving(t *testing.T) {
	store := &recordingStore{}

	status, body := post(t, store, map[string]string{})

	expect.Equal(t, status, http.StatusUnprocessableEntity)
	expect.Equal(t, len(decode[httpjson.ErrorBody](t, body).Fields), len([]string{
		signup.FieldFullName, signup.FieldEmail, signup.FieldPlan, signup.FieldAcceptTerms,
	}))
	expect.Equal(t, store.saved, []signup.Signup(nil))
}

func TestCreateRejectsABodyWithUnknownFields(t *testing.T) {
	status, body := post(t, &recordingStore{}, map[string]string{unknownField: sampleName})

	expect.Equal(t, status, http.StatusBadRequest)
	expect.Equal(t, decode[httpjson.ErrorBody](t, body).Code, httpjson.CodeInvalidBody)
}

func TestCreateReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := post(t, &recordingStore{err: errors.New(storeFailure)}, validRequest())

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, decode[httpjson.ErrorBody](t, body), httpjson.ErrorBody{Code: httpjson.CodeInternal, Message: httpjson.MsgInternal})
}
