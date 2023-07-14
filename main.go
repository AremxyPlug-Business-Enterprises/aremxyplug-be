package main

import (
	"os"

	middleware "github.com/aremxyplug-be/middleware"
	routes "github.com/aremxyplug-be/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
    port := os.Getenv("PORT")

    if port == "" {
        port = "8000"
    }

    router := gin.New()
    router.Use(gin.Logger())
    routes.UserRoutes(router)

    router.Use(middleware.Authentication())


    // API-1
    router.GET("/api-1", func(c *gin.Context) {

        c.JSON(200, gin.H{"success": "Access granted for api-1"})

    })

    // API-2
    router.GET("/api-2", func(c *gin.Context) {
        c.JSON(200, gin.H{"success": "Access granted for api-2"})
    })

    router.Run(":" + port)
}

// cors
func corsConfig() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
        AllowCredentials: true,
    })
}
