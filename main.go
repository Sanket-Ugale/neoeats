// package main

// import (
// 	"os"

// 	"golang-restaurant-management/database"
// 	"golang-restaurant-management/tasks"

// 	middleware "golang-restaurant-management/middleware"
// 	routes "golang-restaurant-management/routes"

// 	"github.com/gin-gonic/gin"

// 	"go.mongodb.org/mongo-driver/mongo"
// )

// var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

// func main() {
// 	go tasks.ProcessEmailQueue()
// 	port := os.Getenv("PORT")

// 	if port == "" {
// 		port = "80"
// 	}
// 	gin.Default()

// 	router := gin.New()
// 	router.Use(gin.Logger())
// 	routes.UserRoutes(router)
// 	routes.HomeRoutes(router)
// 	router.LoadHTMLGlob("templates/*")
// 	router.Use(middleware.Authentication())

// 	routes.FoodRoutes(router)
// 	routes.MenuRoutes(router)
// 	routes.TableRoutes(router)
// 	routes.OrderRoutes(router)
// 	routes.OrderItemRoutes(router)
// 	routes.InvoiceRoutes(router)

//		router.Run("0.0.0.0:" + port)
//	}
package main

import (
	"golang-restaurant-management/database"
	"golang-restaurant-management/middleware"
	"golang-restaurant-management/routes"
	"golang-restaurant-management/tasks"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")
var userCollection *mongo.Collection = database.OpenCollection(database.Client, "users")

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	go tasks.ProcessEmailQueue()
	port := os.Getenv("PORT")

	if port == "" {
		port = "80"
	}

	router := gin.New()
	router.Use(gin.Logger())
	routes.UserRoutes(router)
	routes.HomeRoutes(router)
	router.LoadHTMLGlob("templates/*")
	router.Use(middleware.Authentication())

	routes.FoodRoutes(router)
	routes.MenuRoutes(router)
	routes.TableRoutes(router)
	routes.OrderRoutes(router)
	routes.OrderItemRoutes(router)
	routes.InvoiceRoutes(router)

	router.Run("0.0.0.0:" + port)
}
