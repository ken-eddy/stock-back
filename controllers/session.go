package controllers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/models"
)

func VerifyAuth(c *gin.Context) {
	// Security headers
	// c.Header("Cache-Control", "no-store")
	// c.Header("Pragma", "no-cache")

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("Unauthorized access attempt from IP: %s", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization token required",
			"code":  "MISSING_CREDENTIALS",
		})
		return
	}

	// Database lookup
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User account no longer exists",
				"code":  "USER_NOT_FOUND",
			})
		} else {
			log.Printf("Database error during auth verification: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Could not verify account",
				"code":  "SERVER_ERROR",
			})
		}
		return
	}

	// Successful verification
	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.ID,
		"email":      user.Email,
		"role":       user.Role, // Include if available
		"expires_in": time.Until(time.Unix(int64(c.MustGet("exp").(float64)), 0)),
	})
}
