package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to the Spinride API",
		})
	})

	// API group
	api := r.Group("/api/v1")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})

		api.GET("/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"users": []gin.H{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
			})
		})

		api.GET("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{
				"id":   id,
				"name": "User " + id,
			})
		})

		api.POST("/users", func(c *gin.Context) {
			var json struct {
				Name string `json:"name" binding:"required"`
			}
			if err := c.ShouldBindJSON(&json); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{
				"id":   3,
				"name": json.Name,
			})
		})
	}

	r.Run(":8080")
}
