package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/gorilla/mux"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/semconv"
)

type Service struct {
    ID    uint    `json:"id"`
    Name  string `json:"name"`
    Description string `json:"description"`
    Versions uint `json:"versions"`
}

var service []Service

// Initialize OpenTelemetry
func initTracer() func() {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint()) // Replace with your collector endpoint
    if err != nil {
        panic(err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.ServiceNameKey.String("services"),
        )),
        sdktrace.WithBatcher(exporter),
    )

    otel.SetTracerProvider(tp)

    exporter, err = prometheus.New(prometheus.Config{})
    if err != nil {
        panic(err)
    }

    otel.SetMeterProvider(exporter.MeterProvider())

    return func() {
        _ = exporter.Shutdown(context.Background())
        _ = tp.Shutdown(context.Background())
    }
}

func getServices(w http.ResponseWriter, r *http.Request) {
    ctx, span := otel.GetTracerProvider().Tracer("services").Start(r.Context(), "geServices")
    defer span.End()

    span.AddEvent("Retrieving services from store")
    json.NewEncoder(w).Encode(services)
}

func getService(w http.ResponseWriter, r *http.Request) {
    ctx, span := otel.GetTracerProvider().Tracer("service").Start(r.Context(), "getService")
    defer span.End()

    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "Invalid service ID", http.StatusBadRequest)
        return
    }

    for _, service := range service {
        if service.ID == id {
            json.NewEncoder(w).Encode(service)
            return
        }
    }

    http.Error(w, "Service not found", http.StatusNotFound)
}

func main() {
    defer initTracer()() // Initialize and shutdown OpenTelemetry

    router := mux.NewRouter()
    router.HandleFunc("/services", getServices).Methods("GET")
    router.HandleFunc("/service/{id}", getService).Methods("GET")
    router.Handle("/metrics", otel.Handler()) // Handle Prometheus metrics

    fmt.Println("Starting service on port 8080")
    http.ListenAndServe(":8080", router)
}
