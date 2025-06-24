package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	"github.com/wavlake/api/internal/auth"
	"github.com/wavlake/api/internal/handlers"
	"github.com/wavlake/api/internal/services"
	"google.golang.org/api/option"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable must be set")
	}

	ctx := context.Background()

	// Initialize Firebase
	var firebaseApp *firebase.App
	var err error
	
	// Try to use service account key if available, otherwise use default credentials
	if keyPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_KEY"); keyPath != "" {
		opt := option.WithCredentialsFile(keyPath)
		firebaseApp, err = firebase.NewApp(ctx, nil, opt)
	} else {
		firebaseApp, err = firebase.NewApp(ctx, nil)
	}
	
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	// Initialize Firebase Auth client
	firebaseAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase Auth: %v", err)
	}

	// Initialize Firestore client
	firestoreClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to initialize Firestore: %v", err)
	}
	defer firestoreClient.Close()

	// Initialize services
	userService := services.NewUserService(firestoreClient)

	// Initialize middleware
	firebaseMiddleware := auth.NewFirebaseMiddleware(firebaseAuth)
	dualAuthMiddleware := auth.NewDualAuthMiddleware(firebaseAuth)
	nip98Middleware, err := auth.NewNIP98Middleware(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create NIP-98 middleware: %v", err)
	}

	// Initialize handlers
	authHandlers := handlers.NewAuthHandlers(userService)

	// Set up Gin router
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Heartbeat endpoint (no auth required)
	router.GET("/heartbeat", func(c *gin.Context) {
		handlers.Heartbeat(c.Writer, c.Request)
	})

	// Auth endpoints
	v1 := router.Group("/v1")
	authGroup := v1.Group("/auth")
	{
		// Firebase auth only endpoints
		authGroup.GET("/get-linked-pubkeys", firebaseMiddleware.Middleware(), authHandlers.GetLinkedPubkeys)
		authGroup.POST("/unlink-pubkey", firebaseMiddleware.Middleware(), authHandlers.UnlinkPubkey)
		
		// Dual auth required endpoint
		authGroup.POST("/link-pubkey", dualAuthMiddleware.Middleware(), authHandlers.LinkPubkey)
	}

	// Protected endpoints that require NIP-98 auth
	protectedGroup := v1.Group("/protected")
	protectedGroup.Use(gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Convert back to Gin context
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		c.Next()
	}))))
	{
		// Add NIP-98 protected endpoints here in the future
	}

	// Start server
	log.Printf("Starting server on port %s", port)
	log.Printf("Endpoints available:")
	log.Printf("  GET  /heartbeat")
	log.Printf("  GET  /v1/auth/get-linked-pubkeys (Firebase auth)")
	log.Printf("  POST /v1/auth/unlink-pubkey (Firebase auth)")
	log.Printf("  POST /v1/auth/link-pubkey (Dual auth: Firebase + NIP-98)")

	go func() {
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Server shutdown complete")
}