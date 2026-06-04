package middleware

import (
	"net/http"
	"os"
	"github.com/gin-gonic/gin"
)

func APIkeyAuth() gin.HandlerFunc{
	return func (c *gin.Context){
        clientKey:=c.GetHeader("X-API-Key")
		validKey:=os.Getenv("API_KEY")

		if validKey==""{
			c.Next()
			return 
		}
		if clientKey!=validKey{
			c.AbortWithStatusJSON(http.StatusUnauthorized,gin.H{
				"error":"unauthorized: missing or invalid X-API-Key header",
			})
			return 
		}
		c.Next()
	}
}