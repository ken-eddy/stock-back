package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Sale struct {
	gorm.Model
	BusinessID uint      `json:"business_id" gorm:"not null;index"` // Now linked to a business
	ProductID  uint      `json:"product_id" gorm:"not null;index"`
	Product    Product   `gorm:"foreignKey:ProductID;references:ID"`
	Quantity   int       `json:"quantity" binding:"required"`
	Total      float64   `json:"total" binding:"required"`
	SoldAt     time.Time `json:"sold_at" gorm:"autoCreateTime"`
}
