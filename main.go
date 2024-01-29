package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
)

type Service struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Versions    uint   `json:"versions"`
}

type ServiceRepository interface {
	Services(ctx context.Context) ([]Service, error)
	Service(ctx context.Context, id int) (*Service, error)
}

type ServiceHandler struct {
	repository ServiceRepository
}

func NewServiceHandler(serviceRepo ServiceRepository) ServiceHandler {
	return ServiceHandler{repository: serviceRepo}
}

func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.repository.Services(r.Context())
	if err != nil {
		log.Println("Something bad happened", err)
		http.Error(w, "Badness", http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(services)
}
func (h *ServiceHandler) GetService(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid service ID", http.StatusBadRequest)
		return
	}

	service, err := h.repository.Service(r.Context(), id)

	if err != nil {
		log.Println("Something bad happened", err)
		http.Error(w, "Badness", http.StatusInternalServerError)
	}

	if service != nil {
		json.NewEncoder(w).Encode(service)
		return
	}

	http.Error(w, "Service not found", http.StatusNotFound)
}

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
	router.HandleFunc("/services", handler.GetServices).Methods("GET")
	router.HandleFunc("/service/{id}", handler.GetService).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	fmt.Println("Starting service on port 8080")
	http.ListenAndServe(":8080", router)
}

type AuthenticationMiddleware struct {
	db *sql.DB
}

func NewAuthMiddleware(db *sql.DB) (*AuthenticationMiddleware, error) {
	return &AuthenticationMiddleware{
		db: db,
	}, nil
}

func (amw *AuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		_, span := tracer.Start(r.Context(), "Service")
		defer span.End()
		account, token, _ := r.BasicAuth()
		var found bool
		if err := amw.db.QueryRowContext(r.Context(), "SELECT (count(1)==1) FROM auth WHERE account = ? AND token = ?", account, token).Scan(&found); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Forbidden", http.StatusForbidden)
			}
		}
		if found {
			log.Printf("Authenticated account %s\n", account)
			// Pass down the request to the next middleware (or final handler)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})
}

type InMemoryRepo struct {
	_services []Service
}

func NewInMemoryRepo() (ServiceRepository, error) {
	services := []Service{}

	service := Service{ID: 1, Name: "Locate Us", Description: "Awesomeness is HERE!", Versions: 3}
	services = append(services, service)

	service = Service{ID: 2, Name: "Contact Us", Description: "How can I find you?!", Versions: 2}
	services = append(services, service)

	return &InMemoryRepo{_services: services}, nil
}

func (r *InMemoryRepo) Services(ctx context.Context) ([]Service, error) {
	_, span := tracer.Start(ctx, "Services")
	defer span.End()
	return r._services, nil
}

func (r *InMemoryRepo) Service(ctx context.Context, id int) (*Service, error) {
	_, span := tracer.Start(ctx, "Service")
	defer span.End()

	services, err := r.Services(ctx)
	if err != nil {
		return nil, err
	}

	for _, service := range services {
		if service.ID == id {
			return &service, nil
		}
	}

	return nil, nil
}

type DbRepo struct {
	db *sql.DB
}

func NewDbRepo(db *sql.DB) (*DbRepo, error) {
	// db, err := otelsql.Open("sqlite3", database)
	return &DbRepo{
		db: db,
	}, nil
}

func (r *DbRepo) Services(ctx context.Context) ([]Service, error) {
	_, span := tracer.Start(ctx, "Services")
	defer span.End()
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM services")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []Service{}
	for rows.Next() {
		service := Service{}
		err = rows.Scan(&service.ID, &service.Name, &service.Description, &service.Versions)
		if err != nil {
			return nil, err
		}
		data = append(data, service)
	}
	return data, nil
}

func (r *DbRepo) Service(ctx context.Context, id int) (*Service, error) {
	_, span := tracer.Start(ctx, "Service")
	defer span.End()
	row := r.db.QueryRowContext(ctx, "SELECT * FROM services WHERE id=?", id)

	// Parse row into Activity struct
	service := Service{}
	var err error
	if err = row.Scan(&service.ID, &service.Name, &service.Description, &service.Versions); err == sql.ErrNoRows {
		log.Printf("Id not found")
		return nil, nil
	}

	if err != nil {
		log.Printf("Something bad - %s", err)
		return nil, err
	}
	return &service, nil
}
