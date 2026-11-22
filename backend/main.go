// Command cookie-lab serves the cookie experiment API and the embedded SPA.
package main

import (
	"io/fs"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/JoniDG/cookie-lab/backend/internal/server"
	"github.com/JoniDG/cookie-lab/backend/web"
)

func main() {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg, err := server.LoadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	static, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	r, err := server.NewRouter(cfg, static)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("cookie-lab listening on :%s (COOKIE_MODE=%s)", cfg.Port, cfg.Mode)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
