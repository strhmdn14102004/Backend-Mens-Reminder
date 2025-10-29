package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Error(c *gin.Context, msg string, code int) {
	c.JSON(code, gin.H{"error": msg})
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}
