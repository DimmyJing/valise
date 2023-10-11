package rpc

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
)

func ServeSwaggerUI(ech *echo.Echo, title string, spec []byte, userID string) {
	requestInterceptor := `(req) => { req.headers["Authorization"] = "Bearer ` + userID + `" }`
	//nolint:exhaustruct
	ech.Any("/*", echo.WrapHandler(v5emb.NewHandlerWithConfig(swgui.Config{
		Title:       title,
		SwaggerJSON: "/swagger.json",
		BasePath:    "/",
		HideCurl:    true,
		SettingsUI: map[string]string{
			"tryItOutEnabled":    "true",
			"requestInterceptor": requestInterceptor,
		},
	})))
	ech.GET("/swagger.json", func(c echo.Context) error {
		//nolint:wrapcheck
		return c.JSONBlob(http.StatusOK, spec)
	})
}
