package spa

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed all:dist
var distFS embed.FS

// Register mounts the embedded Vue bundle as a SPA fallback. Any request not
// already handled by an API/asset route will be served from dist/, falling
// back to dist/index.html so client-side routing works.
func Register(app *fiber.App) error {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(sub),
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))
	return nil
}
