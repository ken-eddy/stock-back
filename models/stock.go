package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Stock struct {
	gorm.Model
	BusinessID uint      `json:"business_id" gorm:"not null;index:idx_business_stock"`
	ProductID  uint      `json:"product_id" gorm:"not null;index:idx_product_stock"`
	Product    Product   `gorm:"foreignKey:ProductID;references:ID"`
	Quantity   int       `json:"quantity" binding:"required"`
	AddedAt    time.Time `json:"added_at" gorm:"index;autoCreateTime"`
}
