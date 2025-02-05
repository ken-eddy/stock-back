package controllers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/models"
	"golang.org/x/crypto/bcrypt"
)

// var jwtKey = []byte(os.Getenv("JWT_SECRET")) // Load secret from env

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func CreateUser(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash the password
	hashedPassword, err := HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	input.Password = hashedPassword

	// ✅ Ensure Business ID is assigned if user signs up under an admin
	if input.Role == "user" && input.BusinessID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A user must be assigned to a business by an admin."})
		return
	}

	//allow admin creation without business ID

	if input.Role == "admin" {
		input.BusinessID = 0
	}

	// Save user to database
	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     input.ID,
		"email":       input.Email,
		"role":        input.Role,       // Add this
		"business_id": input.BusinessID, // Add this
		"exp":         expirationTime.Unix(),
	})
	secret := []byte(os.Getenv("JWT_SECRET"))
	tokenString, err := token.SignedString(secret)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Could not create token"})
		return
	}
	// isProduction := os.Getenv("GIN_MODE") == "release"
	// c.SetCookie("token", tokenString, 86400, "/", "", false, false)

	// c.SetCookie("token", tokenString, 86400, "/", "", true, true)
	// isProduction := os.Getenv("GIN_MODE") == "release"
	// secureFlag := isProduction // Secure=true in production, false otherwise

	// c.SetCookie("token", tokenString, 86400, "/", "", secureFlag, true)
	// Replace the existing c.SetCookie() with:
	cookie := fmt.Sprintf(
		"token=%s; Path=/; Max-Age=%d; Secure; HttpOnly; SameSite=None",
		tokenString,
		86400,
	)
	c.Header("Set-Cookie", cookie)

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    input,
	})
}

// login function
func Login(c *gin.Context) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", creds.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !CheckPasswordHash(creds.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour) // Set expiration time

	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"email":       user.Email,
		"role":        user.Role,
		"business_id": user.BusinessID, // 🚀 Added business_id
		"exp":         expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte(os.Getenv("JWT_SECRET"))

	tokenString, err := token.SignedString(secret)
	if err != nil {
		fmt.Println("Token Signing Error:", err) // Debugging log
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create token"})
		return
	}

	// isProduction := os.Getenv("GIN_MODE") == "release"
	// c.SetCookie("token", tokenString, 86400, "/", "", isProduction, true)
	// // c.SetCookie("token", tokenString, 86400, "/", "", true, true)
	cookie := fmt.Sprintf(
		"token=%s; Path=/; Max-Age=%d; Secure; HttpOnly; SameSite=None",
		tokenString,
		86400,
	)
	c.Header("Set-Cookie", cookie)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user_id": user.ID, // Include user_id in response
	})
}

func GetUsers(c *gin.Context) {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetProfile - Fetches the profile of the logged-in user
func GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Fetch the business details
	var business models.Business
	if user.BusinessID != 0 { // Ensure the user is associated with a business
		if err := database.DB.First(&business, user.BusinessID).Error; err != nil {
			// If business not found, return without business details
			c.JSON(http.StatusOK, gin.H{
				"user_id":    user.ID,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      user.Email,
				"role":       user.Role,
				"business":   nil, // No business found
			})
			return
		}
	}

	// Return user details with Business Name
	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.ID,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
		"role":       user.Role,
		"business": gin.H{
			"business_id":   business.ID,
			"business_name": business.BusinessName,
		},
	})
}

func CreateEmployeeUser(c *gin.Context) {
	// Ensure only admins can create employee users
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	// Fetch the admin user
	var admin models.User
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	// Ensure the admin is linked to a business
	if admin.BusinessID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin is not linked to a business"})
		return
	}

	// Parse the input for the new employee user
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash the password
	hashedPassword, err := HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	input.Password = hashedPassword

	// Assign the new user to the admin's business and set role to "employee"
	input.BusinessID = admin.BusinessID
	input.Role = "employee" // Ensure the role is set to "employee"

	// Save the new user to the database
	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Employee user created successfully",
		"user": gin.H{
			"id":          input.ID,
			"first_name":  input.FirstName,
			"last_name":   input.LastName,
			"email":       input.Email,
			"role":        input.Role,
			"business_id": input.BusinessID,
		},
	})
}

// ChangePassword - Allows a user to update their password
func ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	var input struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if old password matches
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect old password"})
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func GetBusinessUsers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	// Fetch logged-in user
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Ensure user is linked to a business
	if user.BusinessID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not assigned to any business"})
		return
	}

	// Fetch all users in the same business
	var businessUsers []models.User
	if err := database.DB.Where("business_id = ?", user.BusinessID).Find(&businessUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch users"})
		return
	}

	// Return only necessary details
	var responseUsers []map[string]interface{}
	for _, u := range businessUsers {
		responseUsers = append(responseUsers, map[string]interface{}{
			"id":         u.ID,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
			"email":      u.Email,
			"role":       u.Role,
		})
	}

	c.JSON(http.StatusOK, responseUsers)
}

func Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// func AuthorizeRoles(roles ...string) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		tokenString := c.GetHeader("Authorization")
// 		if tokenString == "" {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
// 			c.Abort()
// 			return
// 		}

// 		claims := &Claims{}
// 		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
// 			return jwtKey, nil
// 		})
// 		if err != nil || !token.Valid {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
// 			c.Abort()
// 			return
// 		}

// 		for _, role := range roles {
// 			if claims.Role == role {
// 				c.Next()
// 				return
// 			}
// 		}

// 		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
// 		c.Abort()
// 	}
// }
