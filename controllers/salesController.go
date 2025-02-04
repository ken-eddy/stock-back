package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/models"
)

func CreateSale(c *gin.Context) {
	// Get business context
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Business context required"})
		return
	}

	var saleInput struct {
		ProductID uint `json:"product_id" binding:"required"`
		Quantity  int  `json:"quantity" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&saleInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify product belongs to business
	var product models.Product
	if err := database.DB.Where(
		"id = ? AND business_id = ?",
		saleInput.ProductID,
		businessID,
	).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found in your business"})
		return
	}

	if product.Quantity < saleInput.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
		return
	}

	// Create sale with business association
	sale := models.Sale{
		BusinessID: businessID.(uint),
		ProductID:  product.ID,
		Quantity:   saleInput.Quantity,
		Total:      float64(saleInput.Quantity) * product.Price,
		SoldAt:     time.Now(),
	}

	tx := database.DB.Begin()

	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record sale"})
		return
	}

	// Update stock
	product.Quantity -= saleInput.Quantity
	if err := tx.Save(&product).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, sale)
}

func GetSales(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sales []models.Sale
	if err := database.DB.Preload("Product").
		Where("business_id = ?", businessID).
		Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sales"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

func GetProductsForSales(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var products []models.Product
	if err := database.DB.Select("id, name, price, quantity").
		Where("business_id = ?", businessID).
		Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func GetLastFiveSales(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sales []models.Sale
	if err := database.DB.Preload("Product").
		Where("business_id = ?", businessID).
		Order("sold_at DESC").
		Limit(5).
		Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent sales"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

func DeleteSaleRecords(c *gin.Context) {
	businessID, exists := c.Get("business_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := database.DB.Where("business_id = ?", businessID).
		Delete(&models.Sale{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear sales history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sales records cleared successfully"})
}
