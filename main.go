package main

import (
	"context"
	"log"

	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("")

func main() {

	otelShutdown, err := setupOTelSDK(context.Background())

	if err != nil {
		log.Panic("Badness", err)
	}
	defer func() {
		otelShutdown(context.Background())
	}()

	db, err := otelsql.Open("sqlite3", "services.db")
	if err != nil {
		log.Fatalf("Failed to connect to db %s\n", err)
	}

	repo, err := NewDbRepo(db)
	if err != nil {
		log.Panic(err)
	}
	handler := NewServiceHandler(repo)

	authMiddleware, err := NewAuthMiddleware(db)

	if err != nil {
		log.Panicf("Badness %s", err)
	}

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("services"))
	router.Use(authMiddleware.Middleware)
	router.Use(prometheusMiddleware)
	router.HandleFunc("/services", handler.GetServices).Methods("GET")
	router.HandleFunc("/service/{id}", handler.GetService).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	fmt.Println("Starting service on port 8080")
	http.ListenAndServe(":8080", router)
}

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})
)

// prometheusMiddleware implements mux.MiddlewareFunc.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}
