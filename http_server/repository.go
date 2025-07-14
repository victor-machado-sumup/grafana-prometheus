package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

type PaymentMethod string

const (
	PaymentMethod_CreditCard PaymentMethod = "credit"
	PaymentMethod_DebitCard  PaymentMethod = "debit"
	PaymentMethod_Boleto     PaymentMethod = "boleto"
	PaymentMethod_PIX        PaymentMethod = "pix"
)

func IsValidPaymentMethod(value PaymentMethod) bool {
	return value == PaymentMethod_CreditCard ||
		value == PaymentMethod_DebitCard ||
		value == PaymentMethod_Boleto ||
		value == PaymentMethod_PIX
}

func (r *PaymentRepository) SavePayment(ctx context.Context, value float64, paymentMethod PaymentMethod, appName string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO payments (value, payment_method, app_name) VALUES ($1, $2, $3)",
		value, paymentMethod, appName)
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
