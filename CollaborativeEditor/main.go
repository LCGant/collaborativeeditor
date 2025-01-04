// main.go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/LCGant/collaborativeeditor/controllers"
	"github.com/LCGant/collaborativeeditor/models"
)

func main() {
	dsn := "root@tcp(127.0.0.1:3306)/collaborativeeditor?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	if err := db.AutoMigrate(&models.Page{}); err != nil {
		log.Fatalf("Error migrating database: %v", err)
	}

	r := gin.Default()

	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.POST("/access_or_create_page", controllers.AccessOrCreatePageHandler(db))
	r.GET("/editor/:subdomain/*content", controllers.GetPageHandler(db))
	r.POST("/save_page_content", controllers.SavePageContentHandler(db))
	r.GET("/get_page_content/:subdomain", controllers.GetPageContentHandler(db))
	r.GET("/getchildreneditor", controllers.GetChildPagesHandler(db))
	r.GET("/ws/:fullPath", controllers.WebSocketHandler(db))

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
