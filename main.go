// Speedtest LAN - single-binary Go server.
// The web UI (static/) is embedded, so the compiled binary is all you need.
package main

import (
	"bufio"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

var (
	chunk       = make([]byte, 1<<20) // 1 MiB of random, incompressible data
	maxDownload int64
	trustProxy  bool
	ispName     string
	serverName  string
	serverLoc   string
	indexHTML   []byte
)

// loadEnv reads KEY=VALUE lines from path. Real environment variables win.
func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		k, v := strings.TrimSpace(kv[0]), strings.Trim(strings.TrimSpace(kv[1]), `"'`)
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
		}
	}
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int64) int64 {
	if n, err := strconv.ParseInt(getenv(key, ""), 10, 64); err == nil {
		return n
	}
	return def
}

func clientIP(r *http.Request) string {
	if trustProxy {
		for _, h := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
			if v := r.Header.Get(h); v != "" {
				return strings.TrimSpace(strings.Split(v, ",")[0])
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func lanIPs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() && n.IP.To4() != nil {
			out = append(out, n.IP.String())
		}
	}
	return out
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		h.Set("Pragma", "no-cache")
		h.Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST, HEAD, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ip":           clientIP(r),
		"isp":          ispName,
		"server":       serverName,
		"location":     serverLoc,
		"max_download": maxDownload,
		"time":         float64(time.Now().UnixNano()) / 1e9,
	})
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	size := int64(64 << 20)
	if v := r.URL.Query().Get("size"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			size = n
		}
	}
	if size < 1 {
		size = 1
	}
	if size > maxDownload {
		size = maxDownload
	}
	h := w.Header()
	h.Set("Content-Type", "application/octet-stream")
	h.Set("Content-Length", strconv.FormatInt(size, 10))
	h.Set("Content-Encoding", "identity")
	if r.Method == http.MethodHead {
		return
	}
	for sent := int64(0); sent < size; {
		n := int64(len(chunk))
		if size-sent < n {
			n = size - sent
		}
		if _, err := w.Write(chunk[:n]); err != nil {
			return // client closed the connection
		}
		sent += n
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	t0 := time.Now()
	n, _ := io.Copy(io.Discard, r.Body)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"bytes": n, "seconds": time.Since(t0).Seconds()})
}

func main() {
	if exe, err := os.Executable(); err == nil {
		loadEnv(filepath.Join(filepath.Dir(exe), ".env"))
	}
	loadEnv(".env")

	host := getenv("HOST", "0.0.0.0")
	port := getenv("PORT", "8080")
	maxDownload = getenvInt("MAX_DOWNLOAD_MB", 1024) << 20
	trustProxy = getenv("TRUST_PROXY", "0") == "1"
	ispName = getenv("SPEEDTEST_ISP", "Local Network")
	hn, _ := os.Hostname()
	if hn == "" {
		hn = "speedtest"
	}
	serverName = getenv("SPEEDTEST_SERVER_NAME", hn)
	serverLoc = getenv("SPEEDTEST_LOCATION", "LAN")

	if _, err := rand.Read(chunk); err != nil {
		log.Fatal(err)
	}
	var err error
	if indexHTML, err = staticFS.ReadFile("static/index.html"); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") })
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/ping", handlePing)
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/upload", handleUpload)

	srv := &http.Server{
		Addr:              net.JoinHostPort(host, port),
		Handler:           middleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		// No Read/Write timeouts: downloads and uploads are long streams.
	}

	fmt.Printf("Speedtest LAN listening on %s\n", srv.Addr)
	for _, ip := range lanIPs() {
		fmt.Printf("  http://%s:%s\n", ip, port)
	}
	log.Fatal(srv.ListenAndServe())
}
