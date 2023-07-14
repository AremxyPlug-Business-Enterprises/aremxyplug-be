package main

import (
	"os"

	middleware "github.com/aremxyplug-be/middleware"
	routes "github.com/aremxyplug-be/routes"

	// "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
    port := os.Getenv("PORT")

    if port == "" {
        port = "8000"
    }

    router := gin.Default()
    router.Use(corsMiddleware())
    router.Use(gin.Logger())
    routes.UserRoutes(router)
    router.Use(middleware.Authentication())


    
    // // Configure CORS middleware
    // corsConfig := cors.DefaultConfig()
    // corsConfig.AllowOrigins = []string{"*"}
    // corsConfig.AllowCredentials = true
    // corsConfig.AddAllowMethods("OPTIONS")
    // corsConfig.AllowBrowserExtensions = true
    // router.Use(cors.New(corsConfig))


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

// corsMiddleware handles the CORS middleware
func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization", "Origin", "Accept", "Content-Type", "X-Requested-With", "Access-Control-Request-Method", "Access-Control-Request-Headers")
        c.Writer.Header().Set("Access-control-Allow-Credentials", "true")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return

        }
        c.Next()
    }
}

// func CORSMiddleware() gin.HandlerFunc {
//     return func(c *gin.Context) {

//         c.Header("Access-Control-Allow-Origin", "*")
//         c.Header("Access-Control-Allow-Headers", "*")
//         /*
//             c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
//             c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
//             c.Writer.Header().Set("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers")
//             c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH")
//         */

//         if c.Request.Method == "OPTIONS" {
//             c.AbortWithStatus(204)
//             return
//         }

//         c.Next()
//     }
// }
