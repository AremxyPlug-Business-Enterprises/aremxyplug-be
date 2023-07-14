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

    router := gin.Default()
    router.Use(gin.Logger())
    routes.UserRoutes(router)

    router.Use(middleware.Authentication())


    
    // Configure CORS middleware
    corsConfig := cors.DefaultConfig()
    corsConfig.AllowOrigins = []string{"*"}
    corsConfig.AllowCredentials = true
    corsConfig.AddAllowMethods("OPTIONS")
    corsConfig.AllowBrowserExtensions = true
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

// // corsMiddleware handles the CORS middleware
// func corsMiddleware() gin.HandlerFunc {
//     return func(c *gin.Context) {
//         c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
//         c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
//         c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
//         c.Writer.Header().Set("Access-control-Allow-Redirect", "true")
//         c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

//         if c.Request.Method == "OPTIONS" {
//             c.String(http.StatusOK, "ok")
//             return

//         }

//         c.Next()
//     }
// }

