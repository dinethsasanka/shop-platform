package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Order represents an incoming order request.
type Order struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Customer  string    `json:"customer"`
	CreatedAt time.Time `json:"created_at"`
}

var kafkaWriter *kafka.Writer

func main() {
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	topic := getEnv("KAFKA_TOPIC", "order-created")
	port := getEnv("PORT", "8081")

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/orders", handleCreateOrder)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("order-service listening on :%s (kafka broker=%s topic=%s)", port, kafkaBroker, topic)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
		Customer  string `json:"customer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ProductID == "" || req.Quantity <= 0 || req.Customer == "" {
		http.Error(w, "product_id, quantity and customer are required", http.StatusBadRequest)
		return
	}

	order := Order{
		ID:        uuid.NewString(),
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Customer:  req.Customer,
		CreatedAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(order)
	if err != nil {
		http.Error(w, "failed to encode order", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(order.ID),
		Value: payload,
	})
	if err != nil {
		log.Printf("failed to publish order event: %v", err)
		http.Error(w, "failed to publish order event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
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
