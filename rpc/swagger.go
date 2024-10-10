package rpc

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func ServeSwaggerUI(ech *echo.Echo, title string, spec []byte, userID string) {
	//nolint:gofumpt
	var swaggerHTML = `<!DOCTYPE html>
<html>

<head>
    <title>Docs</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>

<body>
    <div id="ui-wrapper-new" data-spec="{{spec}}">
        Loading....
    </div>
</body>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
    var swaggerUIOptions = {
      url: "/swagger/swagger.json",
      dom_id: '#ui-wrapper-new', // Determine what element to load swagger ui
      docExpansion: 'list',
      deepLinking: true, // Enables dynamic deep linking for tags and operations
      filter: true,
      tryItOutEnabled: true,
      showMutatedRequest: false,
      requestSnippetsEnabled: true,
      requestSnippets: { defaultExpanded: false },
      requestInterceptor: (req) => { req.headers["Authorization"] = "Bearer ` + userID + `" },
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      plugins: [
        SwaggerUIBundle.plugins.DownloadUrl
      ],
    }

    var ui = SwaggerUIBundle(swaggerUIOptions)

    /** Export to window for use in custom js */
    window.ui = ui
</script>

</html>`

	ech.GET("/swagger/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, swaggerHTML)
	})
	ech.GET("/swagger/swagger.json", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, spec)
	})
}
