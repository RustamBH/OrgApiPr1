package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"org-api/internal/services"
)

type EmployeeHandler struct {
	service *services.EmployeeService
}

func NewEmployeeHandler(service *services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: service}
}

func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // 405
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/departments/")
	path = strings.Split(path, "/")[0]
	departmentID, err := strconv.Atoi(path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid department ID"})
		return
	}

	var input services.CreateEmployeeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	result, err := h.service.Create(departmentID, &input)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound) // 404
		} else {
			w.WriteHeader(http.StatusBadRequest) // 400
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	json.NewEncoder(w).Encode(result)
}
