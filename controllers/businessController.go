package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jinzhu/gorm"
	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/models"
	"golang.org/x/crypto/bcrypt"
)

// BusinessClaims defines JWT claims for business authentication
type BusinessClaims struct {
	BusinessID   uint   `json:"business_id"`
	BusinessName string `json:"business_name"`
	jwt.RegisteredClaims
}

// CreateBusiness - Allows a logged-in user to create a business
func CreateBusiness(c *gin.Context) {
	// Get authenticated user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Parse input
	var input struct {
		BusinessName string `json:"business_name" binding:"required"`
		Password     string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if business name exists
	var existingBusiness models.Business
	if err := database.DB.Where("business_name = ?", input.BusinessName).First(&existingBusiness).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Business name already exists"})
		return
	}

	// Hash business password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create business
	business := models.Business{
		BusinessName: input.BusinessName,
		Password:     string(hashedPassword),
	}

	if err := database.DB.Create(&business).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create business"})
		return
	}

	// Link user to business
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.BusinessID = business.ID
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link user to business"})
		return
	}

	// Generate new token with business context
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"business_id": user.BusinessID,
		"email":       user.Email,
		"role":        user.Role,
		"exp":         expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// c.SetCookie("token", tokenString, 86400, "/", "", true, true)
	// cookie := fmt.Sprintf(
	// 	"token=%s; Path=/; Max-Age=%d; Secure; HttpOnly; SameSite=None",
	// 	tokenString,
	// 	86400,
	// )
	// c.Header("Set-Cookie", cookie)
	cookie := fmt.Sprintf(
		"token=%s; Path=/; Domain=stock-back-73md.onrender.com; Max-Age=%d; Secure; HttpOnly; SameSite=None",
		tokenString,
		86400,
	)
	c.Header("Set-Cookie", cookie)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Business created successfully",
		"business": gin.H{
			"id":   business.ID,
			"name": business.BusinessName,
		},
	})
}

// ✅ Business Login
func LoginBusiness(c *gin.Context) {
	// 1. Extract user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// 2. Bind input
	var input struct {
		BusinessName string `json:"business_name" binding:"required"`
		Password     string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// 3. Get user with business relationship
	var user models.User
	if err := database.DB.Preload("Business").
		First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// 4. Verify user's business association
	if user.Business == nil || user.Business.BusinessName != input.BusinessName {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not authorized for this business",
		})
		return
	}

	// 5. Validate business password
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Business.Password),
		[]byte(input.Password),
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 6. Generate token with combined claims
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id":       user.ID,
		"business_id":   user.Business.ID,
		"business_name": user.Business.BusinessName,
		"role":          user.Role,
		"exp":           expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// c.SetCookie("token", tokenString, 86400, "/", "", true, true)
	// cookie := fmt.Sprintf(
	// 	"token=%s; Path=/; Max-Age=%d; Secure; HttpOnly; SameSite=None",
	// 	tokenString,
	// 	86400,
	// )
	// c.Header("Set-Cookie", cookie)
	cookie := fmt.Sprintf(
		"token=%s; Path=/; Domain=stock-back-73md.onrender.com; Max-Age=%d; Secure; HttpOnly; SameSite=None",
		tokenString,
		86400,
	)
	c.Header("Set-Cookie", cookie)

	c.JSON(http.StatusOK, gin.H{
		"business": gin.H{
			"id":   user.Business.ID,
			"name": user.Business.BusinessName,
		},
		"expires_at": expirationTime.Unix(),
	})
}

// GetBusinesses - Fetches all businesses with their associated users
func GetBusinesses(c *gin.Context) {
	var businesses []models.Business

	// Use Preload with the correct relationship name
	if err := database.DB.Preload("Users").Find(&businesses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, businesses)
}

func AssignUserToBusiness(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	// Fetch admin user
	var admin models.User
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	// Ensure admin is assigned to a business
	if admin.BusinessID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin is not linked to a business"})
		return
	}

	// Parse request body
	var input struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch target user
	var user models.User
	if err := database.DB.First(&user, input.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// **Assign the business ID to the new user**
	user.BusinessID = admin.BusinessID
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User assigned to business successfully"})
}

// CreateCategory - Allows a business owner to create a category
func CreateCategory(c *gin.Context) {
	// Get business context
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Business context required"})
		return
	}

	// Validate input
	var input struct {
		Name string `json:"name" binding:"required,min=2,max=50"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for existing category in the same business
	var existing models.Category
	err := database.DB.Where(
		"business_id = ? AND LOWER(name) = ?",
		businessID,
		strings.ToLower(input.Name),
	).First(&existing).Error

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Category name already exists in your business",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Create category
	category := models.Category{
		BusinessID: businessID.(uint),
		Name:       strings.TrimSpace(input.Name),
	}

	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}

func GetCategories(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"Error": "Unauthorized"})
	}
	var categories []models.Category
	if err := database.DB.Where("business_id = ?", businessID).Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func GetCategoryProducts(c *gin.Context) {
	categoryID := c.Param("id")
	var products []models.Product
	if err := database.DB.
		Where("category_id = ?", categoryID).
		Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

// GetCategory - Fetches details for a single category based on its ID.
func GetCategory(c *gin.Context) {
	// Get the category ID from the URL parameters
	categoryID := c.Param("id")

	var category models.Category
	// Attempt to find the category by primary key.
	if err := database.DB.First(&category, categoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Return the category details as JSON
	c.JSON(http.StatusOK, category)
}

// ChangeBusinessPassword allows an admin to change the business password.
// It expects a JSON payload with "old_business_password" and "new_business_password".
func ChangeBusinessPassword(c *gin.Context) {
	// Retrieve the business ID from the context.
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business context required"})
		return
	}

	// Bind the JSON input.
	var input struct {
		OldBusinessPassword string `json:"old_business_password" binding:"required"`
		NewBusinessPassword string `json:"new_business_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch the business record using the business ID.
	var business models.Business
	if err := database.DB.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	// Validate the provided old business password.
	if err := bcrypt.CompareHashAndPassword([]byte(business.Password), []byte(input.OldBusinessPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Old business password is incorrect"})
		return
	}

	// Hash the new business password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewBusinessPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// Update the business password and save the record.
	business.Password = string(hashedPassword)
	if err := database.DB.Save(&business).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update business password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Business password updated successfully"})
}

// DeleteCategory - Deletes a category if no products are associated with it
func DeleteCategory(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Business context required"})
		return
	}

	categoryID := c.Param("id")

	// Check if the category exists and belongs to the business
	var category models.Category
	if err := database.DB.Where("id = ? AND business_id = ?", categoryID, businessID).
		First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Check if the category has associated products
	var productCount int64
	if err := database.DB.Model(&models.Product{}).
		Where("category_id = ?", categoryID).
		Count(&productCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check associated products"})
		return
	}

	if productCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete category with associated products"})
		return
	}

	// Delete the category
	if err := database.DB.Delete(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// EditCategory - Updates the name of a category
func EditCategory(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Business context required"})
		return
	}

	categoryID := c.Param("id")
	var input struct {
		Name string `json:"name" binding:"required,min=2,max=50"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the category exists and belongs to the business
	var category models.Category
	if err := database.DB.Where("id = ? AND business_id = ?", categoryID, businessID).
		First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Check if another category with the same name exists in the business
	var existing models.Category
	if err := database.DB.Where("business_id = ? AND LOWER(name) = ? AND id != ?",
		businessID, strings.ToLower(input.Name), categoryID).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Category name already exists in your business"})
		return
	}

	// Update category name
	category.Name = strings.TrimSpace(input.Name)
	if err := database.DB.Save(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully", "category": category})
}
