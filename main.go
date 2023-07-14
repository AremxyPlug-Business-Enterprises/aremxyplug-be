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

    // CORS
    corsConfig := cors.DefaultConfig()
    corsConfig.AllowOrigins = []string{"https://aremxyplug.netlify.app"}
    // To be able to send tokens to the server.
    corsConfig.AllowCredentials = true
    // OPTIONS method for ReactJS
    corsConfig.AddAllowMethods("OPTIONS")
    // Register the middleware
    router.Use(cors.New(corsConfig))
    


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
