package vctx

//nolint:gochecknoglobals
var (
	userIDKey contextKey = "userID"
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
