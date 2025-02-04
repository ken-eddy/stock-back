package middleware

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from cookie
		tokenString, err := c.Cookie("token")
		if err != nil || tokenString == "" {
			fmt.Println("DEBUG: No token found in cookies") // 🔍 Add this line
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
			return
		}
		fmt.Println("DEBUG: Token found:", tokenString) // 🔍 Add this line

		// Parse and validate JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Validate expiration
			if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
				return
			}
			// ✅ Set expiration in context
			if exp, ok := claims["exp"].(float64); ok {
				c.Set("exp", exp)
			}

			// Extract user context
			if userID, ok := claims["user_id"].(float64); ok {
				c.Set("user_id", uint(userID))
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
				return
			}

			if businessID, ok := claims["business_id"].(float64); ok {
				c.Set("business_id", uint(businessID))
			}
			if email, ok := claims["email"].(string); ok {
				c.Set("email", email)
			}
			if role, ok := claims["role"].(string); ok {
				c.Set("role", role)
			}

		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// Proceed to the next handler
		c.Next()
	}
}
