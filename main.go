package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"village-bill/database"
	"village-bill/routes"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "village-bill.db"
	}
	database.InitDB(dbPath)

	r := gin.Default()

	// Static files
	r.Static("/static", "./public/static")
	r.Static("/uploads", "./uploads")
	r.StaticFile("/", "./public/index.html")
	r.StaticFile("/admin.html", "./public/admin.html")

	// Routes
	routes.SetupRoutes(r)

	log.Println("Server is running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
