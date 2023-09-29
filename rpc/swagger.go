package rpc

import (
	"net/http"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
)

func (r *Router) ServeSwaggerUI(spec []byte, userID string) error {
	requestInterceptor := `(req) => { req.headers["Authorization"] = "Bearer ` + userID + `" }`
	//nolint:exhaustruct
	r.mux.Handle("/", v5emb.NewHandlerWithConfig(swgui.Config{
		Title:       r.document.Info.Title,
		SwaggerJSON: "/swagger.json",
		BasePath:    "/",
		HideCurl:    true,
		SettingsUI: map[string]string{
			"tryItOutEnabled":    "true",
			"requestInterceptor": requestInterceptor,
		},
	}))
	r.mux.Handle("/swagger.json", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(spec)
	}))

	return nil
}
