package middleware

import "github.com/gin-gonic/gin"

func Recovery() gin.HandlerFunc {
	// Using Gin's built-in recovery middleware for simplicity,
	// In future, we can replace this with a more sophisticated error handling solution if needed.
	return gin.Recovery()
}
