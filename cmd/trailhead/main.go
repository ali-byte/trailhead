// Command trailhead is Trailhead's single entry point: it will wire the
// Postgres-backed adapter.BookmarkRepository, mount the REST API, and
// (once the SPA build exists) serve the embedded frontend via go:embed -
// see ARCHITECTURE_RFC.md "Package Organization" and "Deferred to Phase F".
//
// At the Phase B gate this file is intentionally a minimal, compiling
// placeholder: it must not import internal/testutil (test doubles are
// never used in production code - see ARCHITECTURE_RFC.md), and the
// internal/adapter/postgres implementation does not exist yet, so there is
// nothing real to wire in yet. A //go:embed directive is not added here
// until web/ has an actual build output to point at - go:embed fails to
// compile against a directory that does not exist on disk.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

// Config holds every environment-derived setting main needs. loadConfig is
// the sole reader of os.Getenv - see go-patterns "Dependency Injection"
// (no other constructor reads the environment directly).
type Config struct {
	Port string // from PORT, default "8080"

	// DatabaseURL from DATABASE_URL is read here at Phase B so the shape
	// of Config is locked, but it is not yet consumed - the Postgres
	// adapter that will use it is a Phase F deliverable.
	DatabaseURL string
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

func main() {
	cfg := loadConfig()

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// TODO(Phase F): wire internal/adapter/postgres.NewRepository(cfg.DatabaseURL)
	// into internal/api handlers, mount them on r, and go:embed the built
	// web/ SPA once it exists.

	addr := ":" + cfg.Port
	log.Printf("trailhead listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("trailhead: %v", err)
	}
}
