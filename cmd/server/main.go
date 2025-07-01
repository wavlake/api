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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/wavlake/api/internal/auth"
	"github.com/wavlake/api/internal/handlers"
	"github.com/wavlake/api/internal/services"
	"github.com/wavlake/api/internal/utils"
	"google.golang.org/api/option"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Println("Warning: GOOGLE_CLOUD_PROJECT environment variable not set")
		// For local development, you might want to set a default or exit gracefully
		projectID = "default-project" // Or handle this appropriately
	}

	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		log.Println("Warning: GCS_BUCKET_NAME environment variable not set")
		// For local development, you might want to set a default or exit gracefully
		bucketName = "default-bucket" // Or handle this appropriately
	}

	tempDir := os.Getenv("TEMP_DIR")
	if tempDir == "" {
		tempDir = "/tmp"
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
	storageService, err := services.NewStorageService(ctx, bucketName)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}
	defer storageService.Close()

	nostrTrackService := services.NewNostrTrackService(firestoreClient, storageService)
	audioProcessor := utils.NewAudioProcessor(tempDir)
	processingService := services.NewProcessingService(storageService, nostrTrackService, audioProcessor, tempDir)

	// Initialize middleware
	firebaseMiddleware := auth.NewFirebaseMiddleware(firebaseAuth)
	dualAuthMiddleware := auth.NewDualAuthMiddleware(firebaseAuth)
	nip98Middleware, err := auth.NewNIP98Middleware(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create NIP-98 middleware: %v", err)
	}

	// Initialize handlers
	authHandlers := handlers.NewAuthHandlers(userService)
	tracksHandler := handlers.NewTracksHandler(nostrTrackService, processingService, audioProcessor)

	// Set up Gin router
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:8080",          // Development
		"http://localhost:3000",          // Alternative dev port
		"http://localhost:8083",          // Another dev port
		"https://wavlake.com",            // Production
		"https://*.wavlake.com",          // Subdomains
		"https://web-wavlake.vercel.app", // Vercel preview deployments
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-Nostr-Authorization",
		"X-Requested-With",
	}
	config.AllowCredentials = true
	router.Use(cors.New(config))

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

		// NIP-98 signature validation only endpoint (no database lookup required)
		authGroup.POST("/check-pubkey-link", gin.WrapH(nip98Middleware.SignatureValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			authHandlers.CheckPubkeyLink(c)
		}))))
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

	// Tracks endpoints
	tracksGroup := v1.Group("/tracks")
	{
		// Public endpoints
		tracksGroup.GET("/:id", tracksHandler.GetTrack)

		// Webhook endpoint for processing notifications
		tracksGroup.POST("/webhook/process", tracksHandler.ProcessTrackWebhook)

		// NIP-98 authenticated endpoints
		tracksGroup.POST("/nostr", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Convert to Gin and call handler
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			// Copy context values from NIP-98 middleware
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.CreateTrackNostr(c)
		}))))

		tracksGroup.GET("/my", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.GetMyTracks(c)
		}))))

		tracksGroup.DELETE("/:id", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.DeleteTrack(c)
		}))))

		// Track status endpoint
		tracksGroup.GET("/:id/status", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.GetTrackStatus(c)
		}))))

		// Manual processing trigger
		tracksGroup.POST("/:id/process", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.TriggerProcessing(c)
		}))))

		// Compression management endpoints
		tracksGroup.POST("/:id/compress", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.RequestCompression(c)
		}))))

		tracksGroup.PUT("/:id/compression-visibility", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.UpdateCompressionVisibility(c)
		}))))

		tracksGroup.GET("/:id/public-versions", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			if pubkey := r.Context().Value("pubkey"); pubkey != nil {
				c.Set("pubkey", pubkey)
			}
			if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
				c.Set("firebase_uid", firebaseUID)
			}
			tracksHandler.GetPublicVersions(c)
		}))))
	}

	// Start server
	log.Printf("Starting server on port %s", port)
	log.Printf("Endpoints available:")
	log.Printf("  GET  /heartbeat")
	log.Printf("  GET  /v1/auth/get-linked-pubkeys (Firebase auth)")
	log.Printf("  POST /v1/auth/unlink-pubkey (Firebase auth)")
	log.Printf("  POST /v1/auth/link-pubkey (Dual auth: Firebase + NIP-98)")
	log.Printf("  POST /v1/auth/check-pubkey-link (NIP-98 signature-only: Check own pubkey link status)")
	log.Printf("  GET  /v1/tracks/:id (Public track info)")
	log.Printf("  POST /v1/tracks/webhook/process (Processing webhook)")
	log.Printf("  POST /v1/tracks/nostr (NIP-98 auth: Create track)")
	log.Printf("  GET  /v1/tracks/my (NIP-98 auth: Get my tracks)")
	log.Printf("  DELETE /v1/tracks/:id (NIP-98 auth: Delete track)")
	log.Printf("  GET  /v1/tracks/:id/status (NIP-98 auth: Get track status)")
	log.Printf("  POST /v1/tracks/:id/process (NIP-98 auth: Trigger processing)")
	log.Printf("  POST /v1/tracks/:id/compress (NIP-98 auth: Request compression versions)")
	log.Printf("  PUT  /v1/tracks/:id/compression-visibility (NIP-98 auth: Update version visibility)")
	log.Printf("  GET  /v1/tracks/:id/public-versions (NIP-98 auth: Get public versions for Nostr)")

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
