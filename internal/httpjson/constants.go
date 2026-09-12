package httpjson

const (
	HeaderContentType = "Content-Type"
	ContentTypeJSON   = "application/json"
	MaxBodyBytes      = 1 << 20
	LogKeyError       = "error"
)

const (
	CodeInvalidBody = "invalid_body"
	CodeValidation  = "validation_failed"
	CodeInternal    = "internal_error"
	MsgInvalidBody  = "request body must hold one JSON object with known fields"
	MsgValidation   = "one or more fields need attention"
	MsgInternal     = "the server could not complete the request"
)

const (
	objectStart      = '{'
	msgWriteFailed   = "response write failed"
	msgRequestFailed = "request failed"
	wrapFormat       = "%w: %w"
)
