package ctx

//nolint:gochecknoglobals
var (
	userIDKey        contextKey = "userID"
	isDevelopmentKey contextKey = "isDevelopment"
	//nolint:gosec
	tokenVerifierKey contextKey = "tokenVerifier"
)

func (c Context) WithUserID(userID string) Context {
	return c.WithValue(userIDKey, userID)
}

func (c Context) UserID() (string, bool) {
	return Value[string](c, userIDKey)
}

func (c Context) MustUserID() string {
	return MustValue[string](c, userIDKey)
}

func (c Context) WithIsDevelopment(isDevelopment bool) Context {
	return c.WithValue(isDevelopmentKey, isDevelopment)
}

func (c Context) IsDevelopment() bool {
	res, ok := Value[bool](c, isDevelopmentKey)

	return ok && res
}

func (c Context) WithTokenVerifier(tokenVerifier func(ctx Context, token string) (string, error)) Context {
	return c.WithValue(tokenVerifierKey, tokenVerifier)
}

func (c Context) TokenVerifier() func(ctx Context, token string) (string, error) {
	res, ok := Value[func(ctx Context, token string) (string, error)](c, tokenVerifierKey)
	if ok {
		return res
	} else {
		return nil
	}
}
