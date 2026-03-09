package middleware

import "github.com/gin-gonic/gin"

func Logger() gin.HandlerFunc {
	// Using Gin's built-in logger for simplicity,
	// In future, we can replace this with a more sophisticated logging solution if needed.
	return gin.Logger()
}
