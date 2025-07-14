package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	port = flag.String("port", "8080", "Port to listen on")
	app  = flag.String("app", "go-app", "Go app name")
)

func main() {
	flag.Parse()

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
