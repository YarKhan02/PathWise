package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed all:assets
var assets embed.FS

// Handler returns the dashboard's HTTP handler: the scan as JSON at /api/scan,
// and the embedded single-page app everywhere else. Unknown non-API paths fall
// back to index.html so the app's client-side routes resolve on a hard refresh.
func Handler(scan Scan) http.Handler {
	ui, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err) // the embed path is a constant; it cannot fail at runtime
	}
	fileServer := http.FileServer(http.FS(ui))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scan)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if f, err := ui.Open(name); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		serveIndex(w, ui)
	})
	return mux
}

// serveIndex writes the SPA entrypoint, for a client-side route with no file.
func serveIndex(w http.ResponseWriter, ui fs.FS) {
	f, err := ui.Open("index.html")
	if err != nil {
		http.Error(w, "dashboard assets are not built", http.StatusNotImplemented)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, f)
}

// Serve runs the dashboard on addr until ctx is cancelled, then shuts down
// cleanly. It binds before reporting the URL so the caller can print a link the
// user can actually open.
func Serve(ctx context.Context, addr string, scan Scan, onReady func(url string)) error {
	srv := &http.Server{Handler: Handler(scan)}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	if onReady != nil {
		onReady("http://" + ln.Addr().String())
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
