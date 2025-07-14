package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	port = flag.String("port", "8080", "Port to listen on")
	app  = flag.String("app", "go-app", "Go app name")
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

func (r *PaymentRepository) SavePayment(ctx context.Context, value float64, appName string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO payments (value, app_name) VALUES ($1, $2)",
		value, appName)
	return err
}

func waitForDB(ctx context.Context, dsn string, maxAttempts int) (*pgxpool.Pool, error) {
	var dbpool *pgxpool.Pool
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Attempting to connect to PostgreSQL (attempt %d/%d)...", attempt, maxAttempts)

		dbpool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			// Try to ping the database
			if err = dbpool.Ping(ctx); err == nil {
				log.Println("Successfully connected to PostgreSQL")
				return dbpool, nil
			}
		}

		if attempt < maxAttempts {
			log.Printf("Failed to connect: %v. Retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}

	return nil, err
}

func main() {
	flag.Parse()

	// Initialize connection pool with retry mechanism
	ctx := context.Background()
	dbpool, err := waitForDB(ctx, "postgres://postgres:postgres@postgres:5432/payments", 12) // 1 minute total wait time
	if err != nil {
		log.Fatalf("Unable to establish connection to database after retries: %v\n", err)
	}
	defer dbpool.Close()

	paymentRepo := NewPaymentRepository(dbpool)
	r := gin.Default()

	// Expose metrics endpoint for Prometheus to scrape
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	type PaymentPayload struct {
		Value float64 `json:"value"`
	}

	r.POST("/payment", func(c *gin.Context) {
		var payload PaymentPayload

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		log.Printf("Received payment with value: %.2f", payload.Value)

		fail := rand.Float64() > 0.7
		if fail {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something wrong",
			})
			return
		}

		// Save payment to database
		err := paymentRepo.SavePayment(c.Request.Context(), payload.Value, *app)
		if err != nil {
			log.Printf("Error saving payment: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error saving payment",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "Payment processed successfully",
			"app":     *app,
			"value":   payload.Value,
		})
	})

	// Run the server
	log.Printf("Starting server on port %s", *port)
	log.Fatal(r.Run(":" + *port))
}
