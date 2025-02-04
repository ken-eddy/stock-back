package models

import (
	"github.com/jinzhu/gorm"
)

type Category struct {
	gorm.Model
	BusinessID uint `json:"business_id" gorm:"not null;index"` // Links category to a business
	// Name       string    `json:"name" gorm:"not null;unique"`
	Name     string    `json:"name" gorm:"not null;uniqueIndex:idx_business_category"` // Composite unique with business
	Products []Product `json:"products" gorm:"foreignKey:CategoryID"`                  // One-to-Many with Products
}
