package main

import (
	"context"
	"fmt"
	"math/rand"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// Using promauto to automatically register the collectors
	taskDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "batch_task_duration_seconds",
		Help: "Duration of the batch task in seconds",
	})

	taskCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "batch_task_total",
		Help: "Total number of batch tasks completed",
	})
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create a new pusher and use the default registry
	pusher := push.New("http://pushgateway:9091", "batch_job").Gatherer(prometheus.DefaultGatherer)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Try one final push before shutting down
				if err := pusher.Push(); err != nil {
					fmt.Printf("final push failed: %v\n", err)
				}
				return
			case <-ticker.C:
				if err := pusher.Push(); err != nil {
					fmt.Printf("Could not push to Pushgateway: %v\n", err)
				}
			}
		}
	}()

	// Simulate a job running every 3 seconds
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Received termination signal. Shutting down...")
			return
		case <-ticker.C:
			// Simulate some work and record metrics
			startTime := time.Now()

			fmt.Println("task started")

			// Simulate random work duration between 1-5 seconds
			workDuration := rand.Float64()*4 + 1
			time.Sleep(time.Duration(workDuration * float64(time.Second)))

			// Set the task duration
			duration := time.Since(startTime).Seconds()
			taskDuration.Set(duration)

			// Increment the counter
			taskCounter.Inc()

			fmt.Printf("task finished %v\n", duration)
		}
	}
}
