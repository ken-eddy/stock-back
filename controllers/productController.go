package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/models"
)

func handleProductError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func GetProducts(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var products []models.Product
	if err := database.DB.Where("business_id = ?", businessID).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

func GetProduct(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var product models.Product
	if err := database.DB.
		Where("business_id = ? AND id = ?", businessID, id).
		First(&product).Error; err != nil {
		handleProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func CreateProduct(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Quantity    int     `json:"quantity"`
		Price       float64 `json:"price"`
		CategoryID  uint    `json:"category_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for existing product
	var existingProduct models.Product
	err := database.DB.Where(
		"business_id = ? AND category_id = ? AND LOWER(name) = ?",
		businessID,
		input.CategoryID,
		strings.ToLower(input.Name),
	).First(&existingProduct).Error

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Product with this name already exists in this category",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()

	// Create product
	product := models.Product{
		BusinessID:  businessID.(uint),
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		Quantity:    input.Quantity,
		Price:       input.Price,
	}

	if err := tx.Create(&product).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create initial stock entry
	stock := models.Stock{
		BusinessID: businessID.(uint),
		ProductID:  product.ID,
		Quantity:   input.Quantity,
		AddedAt:    time.Now(),
	}

	if err := tx.Create(&stock).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product,
	})
}

func UpdateProduct(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var product models.Product
	if err := database.DB.
		Where("business_id = ? AND id = ?", businessID, id).
		First(&product).Error; err != nil {
		handleProductError(c, err)
		return
	}

	var input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Quantity    int     `json:"quantity"`
		Price       float64 `json:"price"`
		CategoryID  uint    `json:"category_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for name conflict
	if input.Name != product.Name || input.CategoryID != product.CategoryID {
		var existing models.Product
		err := database.DB.Where(
			"business_id = ? AND category_id = ? AND LOWER(name) = ? AND id != ?",
			businessID,
			input.CategoryID,
			strings.ToLower(input.Name),
			product.ID,
		).First(&existing).Error

		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Product with this name already exists in this category",
			})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	oldQuantity := product.Quantity
	quantityDiff := input.Quantity - oldQuantity

	tx := database.DB.Begin()

	// Update product
	updateData := models.Product{
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		Quantity:    input.Quantity,
		Price:       input.Price,
	}

	if err := tx.Model(&product).Updates(updateData).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create stock entry if quantity increased
	if quantityDiff > 0 {
		stock := models.Stock{
			BusinessID: businessID.(uint),
			ProductID:  product.ID,
			Quantity:   quantityDiff,
			AddedAt:    time.Now(),
		}

		if err := tx.Create(&stock).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	tx.Commit()

	c.JSON(http.StatusOK, product)
}

// Other functions (DeleteProduct, NumberOfProducts, LowStockItems, etc.) remain the same
// ... [rest of the code remains unchanged] ...

func DeleteProduct(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	id := c.Param("id")
	var product models.Product
	if err := database.DB.
		Where("business_id = ? AND id = ?", user.BusinessID, id).
		First(&product).Error; err != nil {
		handleProductError(c, err)
		return
	}

	if err := database.DB.Delete(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// getting total no. of products
func NumberOfProducts(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"Error": "Unauthorized"})
		return
	}
	var count int64
	var products []models.Product
	if err := database.DB.Where("business_id = ?", businessID).Find(&products).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Errror": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": count})
}

// getting all low-stock items
func LowStockItems(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var products []models.Product
	lowStockThreshold := 10

	if err := database.DB.Where("business_id = ? AND quantity <= ?", businessID, lowStockThreshold).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch low stock items"})
		return
	}
	c.JSON(http.StatusOK, products)
}

// getting number of low stock items
func LowStock(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var products []models.Product
	var count int64
	lowStockThreshold := 10

	if err := database.DB.Where("business_id = ? AND quantity <= ?", businessID, lowStockThreshold).Find(&products).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch low stock items"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lowstock": count})
}

// getting total value of products in the inventory
func TotalValue(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var totalValue float64
	err := database.DB.Raw(
		"SELECT COALESCE(SUM(price * quantity), 0) AS total_value FROM products WHERE business_id = ? AND deleted_at IS NULL",
		businessID,
	).Row().Scan(&totalValue)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"totalValue": totalValue})
}

// deleting all products
func DeleteAll(c *gin.Context) {
	var products []models.Product

	if err := database.DB.Delete(&products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete products"})
		return
	}

	c.JSON(http.StatusOK, "products deleted")
}
