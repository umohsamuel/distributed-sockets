package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/umohsamuel/distributed-sockets/internal/service"
	"github.com/umohsamuel/distributed-sockets/pkg/env"
	"github.com/umohsamuel/distributed-sockets/pkg/response"
)

type Server struct {
	Service     *service.Services
	Engine      *gin.Engine
	Environment *env.EnvironmentVariables
}

func API(services *service.Services, environment *env.EnvironmentVariables) *Server {

	r := &Server{
		Service:     services,
		Engine:      gin.Default(),
		Environment: environment,
	}

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"POST", "GET", "PUT", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept", "User-Agent", "Cache-Control", "Pragma"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	r.Engine.Use(cors.New(config))

	r.Engine.Static("/downloads", "tmp")

	r.Health()

	r.Engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// api := r.Engine.Group("/api/v1")

	return r
}

func (server *Server) Health() {
	server.Engine.GET("/health", func(c *gin.Context) {
		response.NewSuccessResponse("server up!!!", nil, nil).Send(c)
	})
}
