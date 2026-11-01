package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

// route connects a path that clients call on the gateway to the service
// that handles it.
type route struct {
	path    string // path on the gateway, for example /api/orders
	service string // address of the service, for example http://order-service:8081
}

func main() {
	port := getEnv("PORT", "8000")

	// The service addresses come from environment variables, so the same code
	// works with Docker Compose now and with Kubernetes later.
	routes := []route{
		{path: "/api/orders", service: getEnv("ORDER_SERVICE_URL", "http://localhost:8081")},
		{path: "/api/stock", service: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8082")},
		{path: "/api/stats", service: getEnv("ANALYTICS_SERVICE_URL", "http://localhost:8084")},
	}

	router, err := newRouter(routes)
	if err != nil {
		log.Fatalf("invalid route configuration: %v", err)
	}

	log.Printf("gateway listening on :%s", port)
	if err := http.ListenAndServe(":"+port, logRequests(router)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// newRouter creates one reverse proxy for each route.
// Example: a request to /api/orders is forwarded to <order service>/orders.
// The /api prefix is removed before the request is forwarded.
func newRouter(routes []route) (http.Handler, error) {
	mux := http.NewServeMux()

	for _, rt := range routes {
		target, err := url.Parse(rt.service)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return nil, fmt.Errorf("route %s: invalid service address %q", rt.path, rt.service)
		}

		// The reverse proxy (from Go's standard library) sends the request
		// to the service and copies the service's response back to the client.
		proxy := httputil.NewSingleHostReverseProxy(target)

		// If the service cannot be reached, answer with 502 Bad Gateway.
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("cannot reach %s: %v", target.Host, err)
			http.Error(w, "service unavailable", http.StatusBadGateway)
		}

		mux.Handle(rt.path, http.StripPrefix("/api", proxy))
		log.Printf("route %s -> %s", rt.path, rt.service)
	}

	// The gateway's own health check. It is not forwarded to any service.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return mux, nil
}

// statusRecorder wraps the response writer to remember the status code
// that is sent to the client, so it can be written to the log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Unwrap gives Go access to the original response writer
// (needed for features such as flushing streamed responses).
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// logRequests is middleware: it runs around every request. It notes the start
// time, passes the request on, and then logs the method, path, status code
// and duration.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}