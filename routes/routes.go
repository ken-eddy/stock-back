// package routes

// import (
// 	"github.com/gin-gonic/gin"

// 	"github.com/ken-eddy/stockApp/controllers"
// 	"github.com/ken-eddy/stockApp/middleware" // Import the auth middleware
// )

// func SetupRoutes(router *gin.Engine) {
// 	api := router.Group("/api")

// 	// Auth routes should be accessible without authentication
// 	auth := api.Group("/auth")
// 	{
// 		auth.POST("/signup", controllers.CreateUser) // Register user
// 		auth.POST("/login", controllers.Login)       // Login user
// 	}

// 	// Apply authentication middleware AFTER auth routes
// 	// api.Use(middleware.AuthMiddleware()) // 🔐 Require authentication
// 	{
// 		api.POST("/reports", controllers.GenerateReport)

// 		// api.POST("/businesses", middleware.RoleMiddleware("admin"), controllers.CreateBusiness)
// 		// api.POST("/business", controllers.CreateBusiness)
// 		api.POST("/business", middleware.AuthMiddleware(), controllers.CreateBusiness)
// 		api.POST("/business/login", controllers.LoginBusiness)
// 		api.GET("/businesses", controllers.GetBusinesses)
// 		api.POST("/businesses/assign", middleware.RoleMiddleware("admin"), controllers.AssignUserToBusiness)

// 		// api.POST("/categories", middleware.RoleMiddleware("admin"), controllers.CreateCategory)
// 		api.POST("/categories", controllers.CreateCategory)
// 		api.GET("/businesses/:business_id/categories", controllers.GetCategories)
// 	}

// 	products := api.Group("/products")
// 	{
// 		products.GET("/", controllers.GetProducts)
// 		products.GET("/:id", controllers.GetProduct)
// 		products.POST("/", controllers.CreateProduct)
// 		products.PUT("/:id", controllers.UpdateProduct)
// 		products.DELETE("/:id", controllers.DeleteProduct)
// 		products.GET("/total", controllers.NumberOfProducts)
// 		products.GET("/low-stock", controllers.LowStock)
// 		products.GET("/total-value", controllers.TotalValue)
// 		products.GET("/low-stock-items", controllers.LowStockItems)
// 		products.DELETE("", controllers.DeleteAll)
// 	}

// 	category := api.Group("/category")
// 	{
// 		category.GET("", controllers.GetCategory)
// 	}

// 	sales := api.Group("/sales")
// 	{
// 		sales.POST("", controllers.CreateSale)                      // Add a new sale
// 		sales.GET("", controllers.GetSales)                         // Get all sales
// 		sales.GET("/products", controllers.GetProductsForSales)     // Get products for sale
// 		sales.GET("/last-five-sales", controllers.GetLastFiveSales) // Get last 5 sales
// 		sales.DELETE("", controllers.DeleteSaleRecords)
// 	}

// 	users := api.Group("/users")
// 	users.Use(middleware.AuthMiddleware()) // 🔐 Apply authentication middleware
// 	{
// 		users.GET("", controllers.GetUsers)           // List all users (Consider restricting access)
// 		users.GET("/profile", controllers.GetProfile) // 🔹 Profile route
// 		users.GET("/business", controllers.GetBusinessUsers)
// 		users.POST("/changePassword", controllers.ChangePassword)
// 		users.POST("/createEmployee", controllers.CreateEmployeeUser)

//		}
//	}
package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ken-eddy/stockApp/controllers"
	"github.com/ken-eddy/stockApp/middleware"
)

func SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")

	// Public routes
	auth := api.Group("/auth")
	{
		auth.POST("/signup", controllers.CreateUser)
		auth.POST("/login", controllers.Login)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware()) // Apply auth middleware to all routes in this group
	{
		// Business routes
		protected.POST("/business", controllers.CreateBusiness)
		protected.POST("/business/login", controllers.LoginBusiness)
		protected.GET("/businesses", controllers.GetBusinesses)
		protected.POST("/businesses/assign", middleware.RoleMiddleware("admin"), controllers.AssignUserToBusiness)
		protected.POST("/business/changePassword", controllers.ChangeBusinessPassword)

		// Product routes
		products := protected.Group("/products")
		{
			products.GET("/", controllers.GetProducts)
			products.GET("/:id", controllers.GetProduct)
			products.POST("/", controllers.CreateProduct)
			products.PUT("/:id", controllers.UpdateProduct)
			products.DELETE("/:id", controllers.DeleteProduct)
			products.GET("/total", controllers.NumberOfProducts)
			products.GET("/low-stock", controllers.LowStock)
			products.GET("/total-value", controllers.TotalValue)
			products.GET("/low-stock-items", controllers.LowStockItems)
			products.DELETE("", controllers.DeleteAll)
		}

		// Category routes
		protected.POST("/categories", controllers.CreateCategory)
		// protected.GET("/businesses/:business_id/categories", controllers.GetCategories)
		protected.GET("/categories", controllers.GetCategories)
		protected.GET("/categories/:id/products", controllers.GetCategoryProducts)
		protected.GET("/categories/:id", controllers.GetCategory)

		// Sales routes
		sales := protected.Group("/sales")
		{
			sales.POST("", controllers.CreateSale)
			sales.GET("", controllers.GetSales)
			sales.GET("/products", controllers.GetProductsForSales)
			sales.GET("/last-five-sales", controllers.GetLastFiveSales)
			sales.DELETE("", controllers.DeleteSaleRecords)
		}

		// User routes
		users := protected.Group("/users")
		{
			users.GET("", controllers.GetUsers)
			users.GET("/profile", controllers.GetProfile)
			users.GET("/business", controllers.GetBusinessUsers)
			users.POST("/changePassword", controllers.ChangePassword)
			users.POST("/createEmployee", controllers.CreateEmployeeUser)
			users.POST("/logout", controllers.Logout)
		}

		// Reports
		protected.POST("/reports", controllers.GenerateReport)
		protected.GET("/session", controllers.VerifyAuth)
	}
}
