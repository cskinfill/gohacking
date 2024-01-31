package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
