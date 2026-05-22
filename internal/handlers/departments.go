package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"org-api/internal/services"
)

type DepartmentHandler struct {
	service *services.DepartmentService
}

func NewDepartmentHandler(service *services.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *DepartmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input services.CreateDepartmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	result, err := h.service.Create(&input)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
		} else if strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *DepartmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // 405
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/departments/")
	idStr = strings.Split(idStr, "?")[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid department ID"})
		return
	}

	depth := 1
	if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil {
			depth = d
		}
	}

	includeEmployees := true
	if includeStr := r.URL.Query().Get("include_employees"); includeStr != "" {
		if includeStr == "false" {
			includeEmployees = false
		}
	}

	dept, err := h.service.GetByID(id, depth, includeEmployees)
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
	json.NewEncoder(w).Encode(dept)
}

func (h *DepartmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/departments/")
	idStr = strings.Split(idStr, "?")[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid department ID"})
		return
	}

	var input services.UpdateDepartmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	result, err := h.service.Update(id, &input)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound) // 404
		} else if strings.Contains(err.Error(), "cycle") || strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusConflict) // 409
		} else {
			w.WriteHeader(http.StatusBadRequest) // 400
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *DepartmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // 405
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/departments/")
	idStr = strings.Split(idStr, "?")[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid department ID"})
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode != "cascade" && mode != "reassign" {
		w.WriteHeader(http.StatusBadRequest) // 400
		json.NewEncoder(w).Encode(ErrorResponse{Error: "mode must be 'cascade' or 'reassign'"})
		return
	}

	var reassignToDepartmentID *int
	if mode == "reassign" {
		reassignIDStr := r.URL.Query().Get("reassign_to_department_id")
		if reassignIDStr == "" {
			w.WriteHeader(http.StatusBadRequest) // 400
			json.NewEncoder(w).Encode(ErrorResponse{Error: "reassign_to_department_id is required when mode is reassign"})
			return
		}
		if reassignID, err := strconv.Atoi(reassignIDStr); err == nil {
			reassignToDepartmentID = &reassignID
		} else {
			w.WriteHeader(http.StatusBadRequest) // 400
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid reassign_to_department_id"})
			return
		}
	}

	err = h.service.Delete(id, mode, reassignToDepartmentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound) // 404
		} else {
			w.WriteHeader(http.StatusBadRequest) // 400
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
