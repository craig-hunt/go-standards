package middleware

const (
	HeaderAuthorization   = "Authorization"
	HeaderWWWAuthenticate = "WWW-Authenticate"
	BearerScheme          = "Bearer"
	BearerPrefix          = BearerScheme + " "
	CodeUnauthorized      = "unauthorized"
	MsgUnauthorized       = "a valid bearer token is required"
)

const (
	LogKeyMethod      = "method"
	LogKeyPath        = "path"
	LogKeyStatus      = "status"
	LogKeyDurationMS  = "duration_ms"
	LogKeyPanic       = "panic"
	MsgRequestHandled = "request handled"
	MsgPanicRecovered = "panic recovered"
)
