// Command trailhead is Trailhead's single entry point: it wires the
// Postgres-backed adapter.BookmarkRepository, mounts the REST API under
// /api, and serves the embedded frontend SPA at / — see
// ARCHITECTURE_RFC.md "Package Organization" and docs/issues/06-frontend-
// board-add-bar.md "UI Contract" ("cmd/trailhead requirements").
//
// SPA embedding is gated behind the `spa` build tag — see spa_embed.go /
// spa_stub.go. A plain `go build ./cmd/trailhead` (no tag) still produces
// a working, API-only binary; `make build-spa` (or any `-tags spa` build)
// additionally serves the built SPA.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"trailhead/internal/adapter/postgres"
	"trailhead/internal/api"
)

// errSPANotEmbedded is returned by the untagged (!spa) spaFS build — see
// spa_stub.go. Declared here (not there) so it exists identically under
// both the spa and !spa build tags; main.go always compiles regardless
// of the tag, but spa_stub.go/spa_embed.go do not both compile together.
var errSPANotEmbedded = errors.New("spaFS: this binary was built without -tags spa; no SPA is embedded")

// Config holds every environment-derived setting main needs. loadConfig is
// the sole reader of os.Getenv - see go-patterns "Dependency Injection"
// (no other constructor reads the environment directly).
type Config struct {
	Port string // from PORT, default "8080"

	// DatabaseURL from DATABASE_URL selects the Postgres-backed adapter.
	// RunMigrations runs against it before the HTTP server starts
	// listening — see the UI Contract's "cmd/trailhead requirements"
	// (Playwright's webServer.url health-check waits for the port to
	// respond, so fixture-seeding INSERTs can safely assume the schema
	// already exists once Playwright proceeds).
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

	if cfg.DatabaseURL == "" {
		log.Fatal("trailhead: DATABASE_URL is not set")
	}

	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("trailhead: run migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("trailhead: connect to database: %v", err)
	}
	defer pool.Close()

	repo := postgres.New(pool, func() time.Time { return time.Now().UTC() })

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// api.NewRouter registers its routes as absolute paths (/api/...) and
	// owns its own Recoverer/NotFound/MethodNotAllowed - mount it
	// untouched so those absolute paths resolve exactly as it expects.
	r.Handle("/api/*", api.NewRouter(repo))

	r.Handle("/*", newSPAHandler())

	addr := ":" + cfg.Port
	log.Printf("trailhead listening on %s (spa_embedded=%t)", addr, spaEmbedded)

	// http.Server with an explicit ReadHeaderTimeout, not the bare
	// http.ListenAndServe(addr, r) package function - an http.Server with
	// no ReadHeaderTimeout is vulnerable to Slowloris-style connections
	// holding a request's headers open indefinitely (gosec G112 / golangci
	// gosec finding, Phase D CI-fix, 2026-07-07).
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("trailhead: %v", err)
	}
}

// newSPAHandler serves the embedded SPA (see spa_embed.go / spa_stub.go)
// with an index.html fallback for any path that isn't a real static
// asset — standard SPA-serving behavior, so a hard refresh on a future
// client-side route (there is only "/" today, per design.md "Navigation:
// none — the app is one view") never 404s. When the binary was built
// without -tags spa, every request here reports the same clear message
// rather than crashing.
func newSPAHandler() http.Handler {
	assets, err := spaFS()
	if err != nil {
		if !errors.Is(err, errSPANotEmbedded) {
			slog.Error("newSPAHandler: failed to load embedded SPA assets", "error", err)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "trailhead: no SPA embedded in this binary (built without -tags spa)", http.StatusNotImplemented)
		})
	}

	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := r.URL.Path
		if requestPath == "/" {
			requestPath = "/index.html"
		}

		if _, statErr := fs.Stat(assets, requestPath[1:]); statErr != nil {
			// Not a real file under dist/ - serve index.html instead of a
			// bare 404, so any client-side route still loads the app shell.
			indexRequest := r.Clone(r.Context())
			indexRequest.URL.Path = "/"
			fileServer.ServeHTTP(w, indexRequest)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
