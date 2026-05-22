package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"org-api/internal/config"
	"org-api/internal/database"
	"org-api/internal/handlers"
	"org-api/internal/repository"
	"org-api/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	_ = godotenv.Load()

	cfg := config.LoadConfig()

	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	deptRepo := repository.NewDepartmentRepository(db)
	empRepo := repository.NewEmployeeRepository(db)

	deptService := services.NewDepartmentService(deptRepo)
	empService := services.NewEmployeeService(deptRepo, empRepo)

	deptHandler := handlers.NewDepartmentHandler(deptService)
	empHandler := handlers.NewEmployeeHandler(empService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/departments/", deptHandler.Create)
	r.Get("/departments/{id}", deptHandler.Get)
	r.Patch("/departments/{id}", deptHandler.Update)
	r.Delete("/departments/{id}", deptHandler.Delete)
	r.Post("/departments/{id}/employees/", empHandler.Create)

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
