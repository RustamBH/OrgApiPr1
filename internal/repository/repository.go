package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"org-api/internal/models"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) Create(dept *models.Department) error {
	return r.db.Create(dept).Error
}

func (r *DepartmentRepository) FindByID(id int) (*models.Department, error) {
	var dept models.Department
	err := r.db.Preload("Children").First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("department not found: %w", err)
		}
		return nil, err
	}
	return &dept, nil
}

func (r *DepartmentRepository) FindByIDWithChildren(id int, depth int) (*models.Department, error) {
	var dept models.Department
	err := r.db.Preload("Employees").First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("department not found: %w", err)
		}
		return nil, err
	}

	if depth > 0 {
		err = r.loadChildren(&dept, dept.ID, depth-1)
		if err != nil {
			return nil, err
		}
	}

	return &dept, nil
}

func (r *DepartmentRepository) loadChildren(dept *models.Department, parentID int, depth int) error {
	var children []models.Department
	err := r.db.Where("parent_id = ?", parentID).Find(&children).Error
	if err != nil {
		return err
	}

	for i := range children {
		if depth > 0 {
			err = r.loadChildren(&children[i], children[i].ID, depth-1)
			if err != nil {
				return err
			}
		}
		dept.Children = append(dept.Children, children[i])
	}
	return nil
}

func (r *DepartmentRepository) Update(dept *models.Department) error {
	return r.db.Save(dept).Error
}

func (r *DepartmentRepository) Delete(id int) error {
	return r.db.Delete(&models.Department{}, id).Error
}

func (r *DepartmentRepository) FindByParentName(parentID int, name string) (*models.Department, error) {
	var dept models.Department
	query := r.db.Where("LOWER(TRIM(name)) = ?", name)
	if parentID == 0 {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}
	err := query.First(&dept).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dept, nil
}

func (r *DepartmentRepository) ReassignEmployees(fromDepartmentID, toDepartmentID int) error {
	return r.db.Model(&models.Employee{}).
		Where("department_id = ?", fromDepartmentID).
		Update("department_id", toDepartmentID).Error
}

func (r *DepartmentRepository) IsAncestor(ancestorID, descendantID int) (bool, error) {
	if ancestorID == descendantID {
		return true, nil
	}

	var count int64
	err := r.db.Model(&models.Department{}).
		Where("id = ? AND parent_id = ?", descendantID, ancestorID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	var parentID sql.NullInt64
	err = r.db.Model(&models.Department{}).Select("parent_id").Where("id = ?", descendantID).Scan(&parentID).Error
	if err != nil || !parentID.Valid {
		return false, err
	}

	return r.IsAncestor(ancestorID, int(parentID.Int64))
}

type EmployeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) Create(emp *models.Employee) error {
	return r.db.Create(emp).Error
}

func (r *EmployeeRepository) FindByDepartment(departmentID int) ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Where("department_id = ?", departmentID).Order("created_at ASC").Find(&employees).Error
	return employees, err
}
