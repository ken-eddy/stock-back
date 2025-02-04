package database

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/ken-eddy/stockApp/config"

	"github.com/ken-eddy/stockApp/models"
)

var DB *gorm.DB

func ConnectDatabase() {
	cfg := config.LoadConfig()
	connectionString := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName, cfg.DBPassword)

	var err error
	DB, err = gorm.Open("postgres", connectionString)
	if err != nil {
		log.Fatal("Failed to connect to database", err)

	}

	DB = DB.Debug()

	//migrations
	DB.AutoMigrate(&models.Product{}, &models.Stock{}, &models.Sale{}, &models.User{}, &models.Business{}, &models.Category{})
}
