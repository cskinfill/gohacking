package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type InMemoryRepo struct {
	_services []Service
}

func NewInMemoryRepo() ServiceRepository {
	services := []Service{}

	service := Service{ID: 1, Name: "Locate Us", Description: "Awesomeness is HERE!", Versions: 3}
	services = append(services, service)

	service = Service{ID: 2, Name: "Contact Us", Description: "How can I find you?!", Versions: 2}
	services = append(services, service)

	return &InMemoryRepo{_services: services}
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

func TestGetServices(t *testing.T) {
	// Set up router and handler
	repo := NewInMemoryRepo()
	handler := NewServiceHandler(repo)

	router := mux.NewRouter()
	router.HandleFunc("/services", handler.GetServices).Methods("GET")

	// Create a request to simulate a GET request to "/services"
	req, err := http.NewRequest("GET", "/services", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder to capture the response
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Assert that the response was successful (status code 200)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Assert the expected response body (if applicable)
	expectedBody := `[{"id":1,"name":"Locate Us","description":"Awesomeness is HERE!","versions":3},{"id":2,"name":"Contact Us","description":"How can I find you?!","versions":2}]`
	if strings.Trim(rr.Body.String(), "\n") != expectedBody {
		t.Errorf("handler returned unexpected body: \ngot  %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestGetServiceByID(t *testing.T) {
	// Set up router and handler
	handler := NewServiceHandler(NewInMemoryRepo())
	router := mux.NewRouter()
	router.HandleFunc("/service/{id}", handler.GetService).Methods("GET")

	// Create a request to simulate a GET request to "/service/123"
	req, err := http.NewRequest("GET", "/service/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = mux.SetURLVars(req, map[string]string{"id": "1"}) // Set path variable

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Assert that the response was successful
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Assert the expected response body (if applicable)
	expectedBody := `{"id":1,"name":"Locate Us","description":"Awesomeness is HERE!","versions":3}`
	if expectedBody != strings.Trim(rr.Body.String(), "\n") {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}
