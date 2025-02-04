package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"

	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/models"
)

type GenerateReportRequest struct {
	ReportType string `json:"reportType" binding:"required"`
	StartDate  string `json:"startDate" binding:"required"`
	EndDate    string `json:"endDate" binding:"required"`
}

type ReportRow struct {
	Date       string
	Product    string
	Quantity   int
	Price      float64
	TotalValue float64
}

func GenerateReport(c *gin.Context) {
	// Get business context
	businessID, exists := c.Get("business_id")
	fmt.Println("Context keys:", c.Keys)

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Business context required"})
		return
	}

	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid startDate format"})
		return
	}
	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endDate format"})
		return
	}

	// Fetch business-specific data
	rows, title, err := fetchReportData(businessID.(uint), req.ReportType, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate PDF
	pdf := generatePDF(rows, title, req.ReportType, startDate, endDate)

	// Set response headers
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_report.pdf", req.ReportType))
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func fetchReportData(businessID uint, reportType string, startDate, endDate time.Time) ([]ReportRow, string, error) {
	var rows []ReportRow
	var title string

	switch reportType {
	case "sales":
		var sales []models.Sale
		if err := database.DB.Preload("Product").
			Where("business_id = ? AND sold_at BETWEEN ? AND ?", businessID, startDate, endDate).
			Find(&sales).Error; err != nil {
			return nil, "", err
		}
		for _, sale := range sales {
			rows = append(rows, ReportRow{
				Date:       sale.SoldAt.Format("2006-01-02"),
				Product:    sale.Product.Name,
				Quantity:   sale.Quantity,
				Price:      sale.Product.Price,
				TotalValue: sale.Total,
			})
		}
		title = "Sales Report"

	case "current-stock":
		var products []models.Product
		if err := database.DB.
			Where("business_id = ?", businessID).
			Find(&products).Error; err != nil {
			return nil, "", err
		}
		for _, product := range products {
			rows = append(rows, ReportRow{
				Date:       "", // Not applicable for stock report
				Product:    product.Name,
				Quantity:   product.Quantity,
				Price:      product.Price,
				TotalValue: float64(product.Quantity) * product.Price,
			})
		}
		title = "Current Stock Report"

	case "added-stock":
		var stockAdditions []models.Stock
		if err := database.DB.Preload("Product").
			Where("business_id = ? AND added_at BETWEEN ? AND ?", businessID, startDate, endDate).
			Find(&stockAdditions).Error; err != nil {
			return nil, "", err
		}
		for _, addition := range stockAdditions {
			rows = append(rows, ReportRow{
				Date:       addition.AddedAt.Format("2006-01-02"),
				Product:    addition.Product.Name,
				Quantity:   addition.Quantity,
				Price:      addition.Product.Price,
				TotalValue: float64(addition.Quantity) * addition.Product.Price,
			})
		}
		title = "Added Stock Report"

	case "low-stock":
		var products []models.Product
		if err := database.DB.
			Where("business_id = ? AND quantity < ?", businessID, 10).
			Find(&products).Error; err != nil {
			return nil, "", err
		}
		for _, product := range products {
			rows = append(rows, ReportRow{
				Date:       "", // Not applicable for low stock report
				Product:    product.Name,
				Quantity:   product.Quantity,
				Price:      product.Price,
				TotalValue: float64(product.Quantity) * product.Price,
			})
		}
		title = "Low Stock Report"

	default:
		return nil, "", fmt.Errorf("invalid report type")
	}

	return rows, title, nil
}

func generatePDF(rows []ReportRow, title, reportType string, startDate, endDate time.Time) []byte {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	// Report Title
	pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Date Range
	pdf.SetFont("Arial", "", 12)
	if reportType != "current-stock" && reportType != "low-stock" {
		pdf.CellFormat(0, 10, fmt.Sprintf("Date Range: %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")), "", 1, "L", false, 0, "")
		pdf.Ln(5)

		// Only for Sales Report: Calculate total items and total sales value
		if reportType == "sales" {
			var totalItems int
			var totalValue float64
			for _, row := range rows {
				totalItems += row.Quantity
				totalValue += row.TotalValue
			}

			pdf.CellFormat(0, 10, fmt.Sprintf("Total Items Sold: %d", totalItems), "", 1, "L", false, 0, "")
			pdf.CellFormat(0, 10, fmt.Sprintf("Total Sales: ksh %.2f", totalValue), "", 1, "L", false, 0, "")
			pdf.Ln(5)
		}
	}

	// Table Header
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 10, "Date", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 10, "Product", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 10, "Quantity", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 10, "Price", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 10, "Total Value", "1", 1, "C", false, 0, "")

	// Table Rows
	pdf.SetFont("Arial", "", 12)
	for _, row := range rows {
		pdf.CellFormat(40, 10, row.Date, "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 10, row.Product, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("%d", row.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("ksh %.2f", row.Price), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("ksh %.2f", row.TotalValue), "1", 1, "R", false, 0, "")
	}

	// Write PDF to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
