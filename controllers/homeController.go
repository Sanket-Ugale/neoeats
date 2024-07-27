package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Home() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Home Page",
		})
	}
}

func Documentation() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "documentation.html", gin.H{})
	}
}
