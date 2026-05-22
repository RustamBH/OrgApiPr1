package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"org-api/internal/models"
	"org-api/internal/repository"

	"gorm.io/gorm"
)

type DepartmentService struct {
	repo *repository.DepartmentRepository
}

func NewDepartmentService(repo *repository.DepartmentRepository) *DepartmentService {
	return &DepartmentService{repo: repo}
}

type CreateDepartmentInput struct {
	Name     string `json:"name"`
	ParentID int    `json:"parent_id"`
}

type UpdateDepartmentInput struct {
	Name        *string `json:"name"`
	ParentID    int     `json:"parent_id"`
	parentIDSet bool
}

func (u *UpdateDepartmentInput) UnmarshalJSON(data []byte) error {
	aux := struct {
		Name     *string `json:"name"`
		ParentID *int    `json:"parent_id"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.Name = aux.Name
	if aux.ParentID != nil {
		u.ParentID = *aux.ParentID
		u.parentIDSet = true
	}
	return nil
}

func (u *UpdateDepartmentInput) parentIDProvided() bool {
	return u.parentIDSet
}

type CreateDepartmentOutput struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ParentID  int    `json:"parent_id,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (s *DepartmentService) Create(input *CreateDepartmentInput) (*CreateDepartmentOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if len(name) > 200 {
		return nil, errors.New("name too long (max 200 characters)")
	}

	if input.ParentID != 0 {
		_, err := s.repo.FindByID(input.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("parent department not found")
			}
			return nil, err
		}

		existing, err := s.repo.FindByParentName(input.ParentID, name)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("department with this name already exists under this parent")
		}
	}

	dept := &models.Department{
		Name:     name,
		ParentID: input.ParentID,
	}

	if err := s.repo.Create(dept); err != nil {
		return nil, err
	}

	return &CreateDepartmentOutput{
		ID:        dept.ID,
		Name:      dept.Name,
		ParentID:  dept.ParentID,
		CreatedAt: dept.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *DepartmentService) GetByID(id int, depth int, includeEmployees bool) (*models.Department, error) {
	if depth < 0 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}

	dept, err := s.repo.FindByIDWithChildren(id, depth)
	if err != nil {
		return nil, err
	}

	if !includeEmployees {
		dept.Employees = nil
	}

	return dept, nil
}

func (s *DepartmentService) Update(id int, input *UpdateDepartmentInput) (*CreateDepartmentOutput, error) {
	dept, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("department not found")
		}
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, errors.New("name cannot be empty")
		}
		if len(name) > 200 {
			return nil, errors.New("name too long (max 200 characters)")
		}
		dept.Name = name
	}

	if input.parentIDProvided() {
		if input.ParentID == id {
			return nil, errors.New("cannot set department as its own parent")
		}

		if dept.ParentID != input.ParentID {
			if input.ParentID != 0 {
				isAncestor, err := s.repo.IsAncestor(id, input.ParentID)
				if err != nil {
					return nil, err
				}
				if isAncestor {
					return nil, errors.New("cannot create a cycle: department cannot be moved into its own subtree")
				}
			}

			existing, err := s.repo.FindByParentName(input.ParentID, dept.Name)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != id {
				return nil, fmt.Errorf("department with this name already exists under the new parent")
			}
		}

		dept.ParentID = input.ParentID
	}

	if err := s.repo.Update(dept); err != nil {
		return nil, err
	}

	return &CreateDepartmentOutput{
		ID:        dept.ID,
		Name:      dept.Name,
		ParentID:  dept.ParentID,
		CreatedAt: dept.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *DepartmentService) Delete(id int, mode string, reassignToDepartmentID *int) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("department not found")
		}
		return err
	}

	if mode == "reassign" {
		if reassignToDepartmentID == nil {
			return errors.New("reassign_to_department_id is required when mode is reassign")
		}
		if *reassignToDepartmentID == id {
			return errors.New("cannot reassign to the same department being deleted")
		}

		_, err := s.repo.FindByID(*reassignToDepartmentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("target department for reassignment not found")
			}
			return err
		}

		err = s.repo.ReassignEmployees(id, *reassignToDepartmentID)
		if err != nil {
			return err
		}
	}

	return s.repo.Delete(id)
}

type EmployeeService struct {
	deptRepo *repository.DepartmentRepository
	empRepo  *repository.EmployeeRepository
}

func NewEmployeeService(
	deptRepo *repository.DepartmentRepository,
	empRepo *repository.EmployeeRepository,
) *EmployeeService {
	return &EmployeeService{deptRepo: deptRepo, empRepo: empRepo}
}

type CreateEmployeeInput struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type CreateEmployeeOutput struct {
	ID           int     `json:"id"`
	DepartmentID int     `json:"department_id"`
	FullName     string  `json:"full_name"`
	Position     string  `json:"position"`
	HiredAt      *string `json:"hired_at"`
	CreatedAt    string  `json:"created_at"`
}

func (s *EmployeeService) Create(departmentID int, input *CreateEmployeeInput) (*CreateEmployeeOutput, error) {
	_, err := s.deptRepo.FindByID(departmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("department not found")
		}
		return nil, err
	}

	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" {
		return nil, errors.New("full_name cannot be empty")
	}
	if len(fullName) > 200 {
		return nil, errors.New("full_name too long (max 200 characters)")
	}

	position := strings.TrimSpace(input.Position)
	if position == "" {
		return nil, errors.New("position cannot be empty")
	}
	if len(position) > 200 {
		return nil, errors.New("position too long (max 200 characters)")
	}

	emp := &models.Employee{
		DepartmentID: departmentID,
		FullName:     fullName,
		Position:     position,
	}

	if input.HiredAt != nil && *input.HiredAt != "" {
		// Parse date if needed
	}

	if err := s.empRepo.Create(emp); err != nil {
		return nil, err
	}

	hiredAtStr := ""
	if emp.HiredAt != nil {
		hiredAtStr = emp.HiredAt.Format("2006-01-02")
	}

	return &CreateEmployeeOutput{
		ID:           emp.ID,
		DepartmentID: emp.DepartmentID,
		FullName:     emp.FullName,
		Position:     emp.Position,
		HiredAt:      &hiredAtStr,
		CreatedAt:    emp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
