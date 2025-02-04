package models

import (
	"github.com/jinzhu/gorm"
)

type Product struct {
	gorm.Model
	BusinessID  uint     `json:"business_id" gorm:"not null;index"`
	CategoryID  uint     `json:"category_id" gorm:"not null;index"`
	Category    Category `gorm:"foreignKey:CategoryID;references:ID"`
	Name        string   `json:"name" gorm:"not null"`
	Description string   `json:"description"`
	Quantity    int      `json:"quantity" binding:"required"`
	Price       float64  `json:"price" binding:"required"`
	Stocks      []Stock  `gorm:"foreignKey:ProductID"`
	Sales       []Sale   `gorm:"foreignKey:ProductID"`
}
