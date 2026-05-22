package models

import (
	"time"

	"gorm.io/gorm"
)

type Department struct {
	ID        int              `json:"id" gorm:"primaryKey"`
	Name      string           `json:"name" gorm:"not null"`
	ParentID  int              `json:"parent_id,omitempty" gorm:"column:parent_id"`
	CreatedAt time.Time        `json:"created_at"`
	Children  []Department     `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Employees []Employee       `json:"employees,omitempty" gorm:"foreignKey:DepartmentID"`
}

func (d *Department) BeforeCreate(tx *gorm.DB) error {
	if d.ParentID == 0 {
		tx.Statement.SetColumn("parent_id", nil)
	}
	return nil
}

func (d *Department) BeforeUpdate(tx *gorm.DB) error {
	if d.ParentID == 0 {
		tx.Statement.SetColumn("parent_id", nil)
	}
	return nil
}

type Employee struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	DepartmentID int       `json:"department_id" gorm:"not null"`
	FullName     string    `json:"full_name" gorm:"not null"`
	Position     string    `json:"position" gorm:"not null"`
	HiredAt      *time.Time `json:"hired_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
