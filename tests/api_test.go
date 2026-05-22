package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"org-api/internal/config"
	"org-api/internal/database"
	"org-api/internal/handlers"
	"org-api/internal/repository"
	"org-api/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var deptService *services.DepartmentService
var empService *services.EmployeeService
var router *chi.Mux

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "host=localhost user=postgres password=postgres dbname=org_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}

	sqlDB.Exec(`
		DROP TABLE IF EXISTS employees;
		DROP TABLE IF EXISTS departments;
		CREATE TABLE departments (
			id SERIAL PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			parent_id INTEGER REFERENCES departments(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX idx_departments_parent_name ON departments(parent_id, LOWER(TRIM(name)));
		CREATE TABLE employees (
			id SERIAL PRIMARY KEY,
			department_id INTEGER NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
			full_name VARCHAR(200) NOT NULL,
			position VARCHAR(200) NOT NULL,
			hired_at DATE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_employees_department_id ON employees(department_id);
	`)

	return db
}

func setup(t *testing.T) {
	cfg := &config.Config{}
	db = setupTestDB(t)

	deptRepo := repository.NewDepartmentRepository(db)
	empRepo := repository.NewEmployeeRepository(db)

	deptService = services.NewDepartmentService(deptRepo)
	empService = services.NewEmployeeService(deptRepo, empRepo)

	deptHandler := handlers.NewDepartmentHandler(deptService)
	empHandler := handlers.NewEmployeeHandler(empService)

	router = chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Post("/departments/", deptHandler.Create)
	router.Get("/departments/{id}", deptHandler.Get)
	router.Patch("/departments/{id}", deptHandler.Update)
	router.Delete("/departments/{id}", deptHandler.Delete)
	router.Post("/departments/{id}/employees/", empHandler.Create)
}

func teardown(t *testing.T) {
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.Exec("DROP TABLE IF EXISTS employees; DROP TABLE IF EXISTS departments;")
}

func TestCreateDepartment(t *testing.T) {
	setup(t)
	defer teardown(t)

	payload := `{"name": "Engineering"}`
	req, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "Engineering", response["name"])
	assert.NotNil(t, response["id"])
}

func TestCreateDepartmentWithParent(t *testing.T) {
	setup(t)
	defer teardown(t)

	parentPayload := `{"name": "IT"}`
	parentReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(parentPayload))
	parentRR := httptest.NewRecorder()
	router.ServeHTTP(parentRR, parentReq)

	var parent map[string]interface{}
	json.Unmarshal(parentRR.Body.Bytes(), &parent)

	payload := `{"name": "Backend", "parent_id": ` + string(intToJSON(parent["id"])) + `}`
	req, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "Backend", response["name"])
}

func TestCreateDepartmentDuplicateName(t *testing.T) {
	setup(t)
	defer teardown(t)

	payload1 := `{"name": "Engineering"}`
	req1, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload1))
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)

	assert.Equal(t, http.StatusCreated, rr1.Code)

	payload2 := `{"name": "Engineering"}`
	req2, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload2))
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusConflict, rr2.Code)
}

func TestCreateDepartmentEmptyName(t *testing.T) {
	setup(t)
	defer teardown(t)

	payload := `{"name": ""}`
	req, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetDepartment(t *testing.T) {
	setup(t)
	defer teardown(t)

	payload := `{"name": "HR"}`
	req, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	req, _ = http.NewRequest("GET", "/departments/"+string(rune('0'+id)), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreateEmployee(t *testing.T) {
	setup(t)
	defer teardown(t)

	deptPayload := `{"name": "Sales"}`
	deptReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(deptPayload))
	deptRR := httptest.NewRecorder()
	router.ServeHTTP(deptRR, deptReq)

	var dept map[string]interface{}
	json.Unmarshal(deptRR.Body.Bytes(), &dept)
	deptID := int(dept["id"].(float64))

	empPayload := `{"full_name": "John Doe", "position": "Manager"}`
	empReq, _ := http.NewRequest("POST", "/departments/"+string(rune('0'+deptID))+"/employees/", bytes.NewBufferString(empPayload))
	empRR := httptest.NewRecorder()
	router.ServeHTTP(empReq, empRR)

	assert.Equal(t, http.StatusCreated, empRR.Code)

	var response map[string]interface{}
	json.Unmarshal(empRR.Body.Bytes(), &response)
	assert.Equal(t, "John Doe", response["full_name"])
	assert.Equal(t, "Manager", response["position"])
}

func TestCreateEmployeeNonExistentDepartment(t *testing.T) {
	setup(t)
	defer teardown(t)

	payload := `{"full_name": "Jane Doe", "position": "Developer"}`
	req, _ := http.NewRequest("POST", "/departments/9999/employees/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteDepartmentCascade(t *testing.T) {
	setup(t)
	defer teardown(t)

	deptPayload := `{"name": "Test Dept"}`
	deptReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(deptPayload))
	deptRR := httptest.NewRecorder()
	router.ServeHTTP(deptReq, deptRR)

	var dept map[string]interface{}
	json.Unmarshal(deptRR.Body.Bytes(), &dept)
	deptID := int(dept["id"].(float64))

	empPayload := `{"full_name": "John Doe", "position": "Manager"}`
	empReq, _ := http.NewRequest("POST", "/departments/"+string(rune('0'+deptID))+"/employees/", bytes.NewBufferString(empPayload))
	empRR := httptest.NewRecorder()
	router.ServeHTTP(empReq, empRR)

	assert.Equal(t, http.StatusCreated, empRR.Code)

	deleteReq, _ := http.NewRequest("DELETE", "/departments/"+string(rune('0'+deptID))+"?mode=cascade", nil)
	deleteRR := httptest.NewRecorder()
	router.ServeHTTP(deleteReq, deleteRR)

	assert.Equal(t, http.StatusNoContent, deleteRR.Code)
}

func TestDeleteDepartmentReassign(t *testing.T) {
	setup(t)
	defer teardown(t)

	targetDeptPayload := `{"name": "Target Dept"}`
	targetDeptReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(targetDeptPayload))
	targetDeptRR := httptest.NewRecorder()
	router.ServeHTTP(targetDeptReq, targetDeptRR)

	var targetDept map[string]interface{}
	json.Unmarshal(targetDeptRR.Body.Bytes(), &targetDept)
	targetDeptID := int(targetDept["id"].(float64))

	sourceDeptPayload := `{"name": "Source Dept"}`
	sourceDeptReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(sourceDeptPayload))
	sourceDeptRR := httptest.NewRecorder()
	router.ServeHTTP(sourceDeptReq, sourceDeptRR)

	var sourceDept map[string]interface{}
	json.Unmarshal(sourceDeptRR.Body.Bytes(), &sourceDept)
	sourceDeptID := int(sourceDept["id"].(float64))

	empPayload := `{"full_name": "John Doe", "position": "Manager"}`
	empReq, _ := http.NewRequest("POST", "/departments/"+string(rune('0'+sourceDeptID))+"/employees/", bytes.NewBufferString(empPayload))
	empRR := httptest.NewRecorder()
	router.ServeHTTP(empReq, empRR)

	deleteReq, _ := http.NewRequest("DELETE", "/departments/"+string(rune('0'+sourceDeptID))+"?mode=reassign&reassign_to_department_id="+string(rune('0'+targetDeptID)), nil)
	deleteRR := httptest.NewRecorder()
	router.ServeHTTP(deleteReq, deleteRR)

	assert.Equal(t, http.StatusNoContent, deleteRR.Code)
}

func TestUpdateDepartment(t *testing.T) {
	setup(t)
	defer teardown(t)

	payload := `{"name": "Old Name"}`
	req, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	updatePayload := `{"name": "New Name"}`
	updateReq, _ := http.NewRequest("PATCH", "/departments/"+string(rune('0'+id)), bytes.NewBufferString(updatePayload))
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateReq, updateRR)

	assert.Equal(t, http.StatusOK, updateRR.Code)

	var response map[string]interface{}
	json.Unmarshal(updateRR.Body.Bytes(), &response)
	assert.Equal(t, "New Name", response["name"])
}

func TestUpdateDepartmentCycle(t *testing.T) {
	setup(t)
	defer teardown(t)

	parentPayload := `{"name": "Parent"}`
	parentReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(parentPayload))
	parentRR := httptest.NewRecorder()
	router.ServeHTTP(parentReq, parentRR)

	var parent map[string]interface{}
	json.Unmarshal(parentRR.Body.Bytes(), &parent)
	parentID := int(parent["id"].(float64))

	childPayload := `{"name": "Child", "parent_id": ` + string(intToJSON(parentID)) + `}`
	childReq, _ := http.NewRequest("POST", "/departments/", bytes.NewBufferString(childPayload))
	childRR := httptest.NewRecorder()
	router.ServeHTTP(childReq, childRR)

	var child map[string]interface{}
	json.Unmarshal(childRR.Body.Bytes(), &child)
	childID := int(child["id"].(float64))

	updatePayload := `{"parent_id": ` + string(intToJSON(childID)) + `}`
	updateReq, _ := http.NewRequest("PATCH", "/departments/"+string(rune('0'+parentID)), bytes.NewBufferString(updatePayload))
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateReq, updateRR)

	assert.Equal(t, http.StatusConflict, updateRR.Code)
}

func intToJSON(i int) []byte {
	return []byte(string(rune('0' + i)))
}

func TestMain(m *testing.M) {
	cfg := config.LoadConfig()
	var err error
	db, err = database.NewDB(cfg)
	if err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	m.Run()
}
