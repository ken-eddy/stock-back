// package main

// import (
// 	"log"

// 	"github.com/gin-contrib/cors"
// 	"github.com/gin-gonic/gin"
// 	"github.com/joho/godotenv"

// 	"github.com/ken-eddy/stockApp/database"
// 	"github.com/ken-eddy/stockApp/routes"
// )

// func main() {

// 	// Load environment variables
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Fatal("Error loading .env file")
// 	}

// 	//connecting to database
// 	database.ConnectDatabase()
// 	defer database.DB.Close()

// 	//initialize Gin router
// 	router := gin.Default()

// 	router.Use(cors.Default())

// 	//setting up routes
// 	routes.SetupRoutes(router)

//		router.Run(":8080")
//	}
package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/ken-eddy/stockApp/database"
	"github.com/ken-eddy/stockApp/routes"
)

func main() {
	// Load environment variables
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET is not set in .env file")
	}

	// Connect to database
	database.ConnectDatabase()
	defer database.DB.Close()

	// Initialize Gin router
	router := gin.Default()

	// 🔥 FIX: Configure CORS properly
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // Allow frontend
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Setup routes
	routes.SetupRoutes(router)

	// Start server
	router.Run(":8080")
}
