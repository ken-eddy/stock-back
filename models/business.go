package models

import "github.com/jinzhu/gorm"

type Business struct {
	gorm.Model
	BusinessName string     `json:"business_name" gorm:"not null;unique"`
	Password     string     `json:"password" binding:"required" gorm:"not null"`
	Users        []*User    `json:"-" gorm:"foreignKey:BusinessID" ` // 🔥 Use pointer slice
	Products     []*Product `json:"-" gorm:"foreignKey:BusinessID"`
	Stock        []*Stock   `json:"-" gorm:"foreignKey:BusinessID"`
	Sales        []*Sale    `json:"-" gorm:"foreignKey:BusinessID"`
}
