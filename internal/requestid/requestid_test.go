package requestid

import (
	"context"
	"strings"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
)

func TestFromReturnsTheIdentifierStoredWithWith(t *testing.T) {
	ctx := With(context.Background(), suppliedID)

	expect.Equal(t, From(ctx), suppliedID)
}

func TestFromReturnsEmptyWhenNoIdentifierWasStored(t *testing.T) {
	expect.Equal(t, From(context.Background()), "")
}

func TestAttrCarriesTheIdentifierUnderTheSharedKey(t *testing.T) {
	attr := Attr(With(context.Background(), suppliedID))

	expect.Equal(t, attr.Key, LogKey)
	expect.Equal(t, attr.Value.String(), suppliedID)
}

func TestNewGeneratesDistinctIdentifiersOfAFixedLength(t *testing.T) {
	first, second := New(), New()

	expect.Equal(t, len(first), generatedLength)
	expect.True(t, first != second, distinctIDsReason)
}

func TestAcceptKeepsASuppliedIdentifier(t *testing.T) {
	expect.Equal(t, Accept(suppliedID), suppliedID)
}

func TestAcceptKeepsAnIdentifierAtTheMaximumLength(t *testing.T) {
	longest := strings.Repeat(fillerCharacter, MaxLength)

	expect.Equal(t, Accept(longest), longest)
}

func TestAcceptReplacesAnOversizedIdentifier(t *testing.T) {
	oversized := strings.Repeat(fillerCharacter, MaxLength+1)

	expect.Equal(t, len(Accept(oversized)), generatedLength)
}

func TestAcceptGeneratesAnIdentifierWhenNoneWasSupplied(t *testing.T) {
	expect.Equal(t, len(Accept("")), generatedLength)
}
