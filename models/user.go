package models

import (
	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model
	FirstName  string    `json:"first_name" binding:"required"`
	LastName   string    `json:"last_name" binding:"required"`
	Email      string    `json:"email" binding:"required" gorm:"unique;not null"`
	Password   string    `json:"password" binding:"required" gorm:"not null"`
	Role       string    `json:"role" gorm:"default:'user'"`
	BusinessID uint      `json:"business_id" gorm:"index"`
	Business   *Business `json:"-" gorm:"foreignKey:BusinessID"` // 🔥 Use pointer
}
