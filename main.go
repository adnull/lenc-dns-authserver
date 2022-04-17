package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.Default()
	router2 := gin.Default()

	done := make(chan int)

	// Connect to database or panic

	router.GET("/api/", func(c *gin.Context) {
		c.String(200, "Hello")
	})

	router2.GET("/api/", func(c *gin.Context) {
		c.String(200, "Hello 2")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go router.Run(":" + port)
	go router2.Run(":8081")

	<-done
}
