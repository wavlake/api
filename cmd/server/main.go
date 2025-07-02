package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/wavlake/api/internal/auth"
	"github.com/wavlake/api/internal/handlers"
	"github.com/wavlake/api/internal/services"
	"github.com/wavlake/api/internal/utils"
	"google.golang.org/api/option"
)

// getEnvAsInt returns an environment variable as an integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

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

	// Initialize PostgreSQL connection (optional)
	var postgresService services.PostgresServiceInterface
	pgConnStr := os.Getenv("PROD_POSTGRES_CONNECTION_STRING_RO")
	if pgConnStr != "" {
		maxOpenConns := getEnvAsInt("POSTGRES_MAX_CONNECTIONS", 10)
		maxIdleConns := getEnvAsInt("POSTGRES_MAX_IDLE_CONNECTIONS", 5)

		db, err := sql.Open("postgres", pgConnStr)
		if err != nil {
			log.Fatalf("Failed to open PostgreSQL connection: %v", err)
		}
		defer db.Close()

		// Configure connection pool
		db.SetMaxOpenConns(maxOpenConns)
		db.SetMaxIdleConns(maxIdleConns)
		db.SetConnMaxLifetime(time.Hour)

		// Test connection
		if err := db.PingContext(ctx); err != nil {
			log.Printf("PostgreSQL connection test failed: %v", err)
		} else {
			postgresService = services.NewPostgresService(db)
			log.Println("PostgreSQL connection established successfully")
		}
	} else {
		log.Println("PostgreSQL connection string not provided, skipping PostgreSQL setup")
	}

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

	// Initialize legacy handler if PostgreSQL is available
	var legacyHandler *handlers.LegacyHandler
	if postgresService != nil {
		legacyHandler = handlers.NewLegacyHandler(postgresService)
	}

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

	// Legacy endpoints (NIP-98 auth required, PostgreSQL-backed)
	if legacyHandler != nil {
		legacyGroup := v1.Group("/legacy")
		{
			legacyGroup.GET("/metadata", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				if pubkey := r.Context().Value("pubkey"); pubkey != nil {
					c.Set("pubkey", pubkey)
				}
				if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
					c.Set("firebase_uid", firebaseUID)
				}
				legacyHandler.GetUserMetadata(c)
			}))))

			legacyGroup.GET("/tracks", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				if pubkey := r.Context().Value("pubkey"); pubkey != nil {
					c.Set("pubkey", pubkey)
				}
				if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
					c.Set("firebase_uid", firebaseUID)
				}
				legacyHandler.GetUserTracks(c)
			}))))

			legacyGroup.GET("/artists", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				if pubkey := r.Context().Value("pubkey"); pubkey != nil {
					c.Set("pubkey", pubkey)
				}
				if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
					c.Set("firebase_uid", firebaseUID)
				}
				legacyHandler.GetUserArtists(c)
			}))))

			legacyGroup.GET("/albums", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				if pubkey := r.Context().Value("pubkey"); pubkey != nil {
					c.Set("pubkey", pubkey)
				}
				if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
					c.Set("firebase_uid", firebaseUID)
				}
				legacyHandler.GetUserAlbums(c)
			}))))

			legacyGroup.GET("/artists/:artist_id/tracks", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				if pubkey := r.Context().Value("pubkey"); pubkey != nil {
					c.Set("pubkey", pubkey)
				}
				if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
					c.Set("firebase_uid", firebaseUID)
				}
				legacyHandler.GetTracksByArtist(c)
			}))))

			legacyGroup.GET("/albums/:album_id/tracks", gin.WrapH(nip98Middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				if pubkey := r.Context().Value("pubkey"); pubkey != nil {
					c.Set("pubkey", pubkey)
				}
				if firebaseUID := r.Context().Value("firebase_uid"); firebaseUID != nil {
					c.Set("firebase_uid", firebaseUID)
				}
				legacyHandler.GetTracksByAlbum(c)
			}))))
		}
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

	if legacyHandler != nil {
		log.Printf("  GET  /v1/legacy/metadata (NIP-98 auth: Get all user metadata from legacy system)")
		log.Printf("  GET  /v1/legacy/tracks (NIP-98 auth: Get user tracks from legacy system)")
		log.Printf("  GET  /v1/legacy/artists (NIP-98 auth: Get user artists from legacy system)")
		log.Printf("  GET  /v1/legacy/albums (NIP-98 auth: Get user albums from legacy system)")
		log.Printf("  GET  /v1/legacy/artists/:artist_id/tracks (NIP-98 auth: Get tracks by artist)")
		log.Printf("  GET  /v1/legacy/albums/:album_id/tracks (NIP-98 auth: Get tracks by album)")
	}

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
