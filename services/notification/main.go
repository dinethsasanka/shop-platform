package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/segmentio/kafka-go"
)

// StockEvent mirrors what the inventory-service publishes.
type StockEvent struct {
	OrderID      string `json:"order_id"`
	ProductID    string `json:"product_id"`
	Status       string `json:"status"` // "stock-updated" or "out-of-stock"
	RemainingQty int    `json:"remaining_qty"`
}

func main() {
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	stockTopic := getEnv("STOCK_TOPIC", "stock-updated")
	port := getEnv("PORT", "8083")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    stockTopic,
		GroupID:  "notification-service",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	go consumeStockEvents(reader)

	// Minimal HTTP server - just so Docker/Kubernetes can health-check this service.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)

	log.Printf("notification-service listening on :%s (consuming %s)", port, stockTopic)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func consumeStockEvents(reader *kafka.Reader) {
	ctx := context.Background()
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading stock message: %v", err)
			continue
		}

		var event StockEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("error decoding stock message: %v", err)
			continue
		}

		notify(event)
	}
}

// notify decides what, if anything, to tell a human - and "sends" it.
// For now this just prints. Later this is where AWS SES/SNS calls go.
func notify(event StockEvent) {
	switch event.Status {
	case "out-of-stock":
		log.Printf("[ALERT] Out of stock! product=%s order=%s - notifying ops team (simulated email)", event.ProductID, event.OrderID)
	case "stock-updated":
		log.Printf("[INFO] Order %s confirmed for product %s - %d units remaining (simulated customer email)", event.OrderID, event.ProductID, event.RemainingQty)
	default:
		log.Printf("[WARN] unrecognized stock event status: %s", event.Status)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}