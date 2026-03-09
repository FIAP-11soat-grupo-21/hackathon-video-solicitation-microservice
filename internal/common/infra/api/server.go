package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	"video_solicitation_microservice/internal/common/infra/api/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.New()

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.Cors())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	return router
}

func StartServer(ctx context.Context, router *gin.Engine, host, port string) {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: router,
	}

	notifyCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("HTTP server starting on %s:%s", host, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-notifyCtx.Done()
	log.Println("Shutting down HTTP server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server forced shutdown: %v", err)
	}

	log.Println("HTTP server stopped gracefully")
}
