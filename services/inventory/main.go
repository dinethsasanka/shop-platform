package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// OrderEvent mirrors the JSON the order-service publishes.
type OrderEvent struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Customer  string    `json:"customer"`
	CreatedAt time.Time `json:"created_at"`
}

// StockEvent is what we publish after processing an order.
type StockEvent struct {
	OrderID      string `json:"order_id"`
	ProductID    string `json:"product_id"`
	Status       string `json:"status"` // "stock-updated" or "out-of-stock"
	RemainingQty int    `json:"remaining_qty"`
}

// stockStore is a simple thread-safe in-memory stock tracker.
type stockStore struct {
	mu    sync.Mutex
	stock map[string]int
}

func newStockStore() *stockStore {
	return &stockStore{
		stock: map[string]int{
			"p123": 10,
			"p456": 5,
			"p789": 0,
		},
	}
}

func (s *stockStore) reserve(productID string, qty int) (remaining int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.stock[productID]
	if !exists || current < qty {
		return current, false
	}
	s.stock[productID] = current - qty
	return s.stock[productID], true
}

func (s *stockStore) snapshot() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.stock))
	for k, v := range s.stock {
		out[k] = v
	}
	return out
}

var store = newStockStore()

func main() {
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	orderTopic := getEnv("ORDER_TOPIC", "order-created")
	stockTopic := getEnv("STOCK_TOPIC", "stock-updated")
	port := getEnv("PORT", "8082")

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(kafkaBroker),
		Topic:                  stockTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    orderTopic,
		GroupID:  "inventory-service",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	go consumeOrders(reader, writer)

	mux := http.NewServeMux()
	mux.HandleFunc("/stock", handleStock)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("inventory-service listening on :%s (consuming %s, publishing %s)", port, orderTopic, stockTopic)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func consumeOrders(reader *kafka.Reader, writer *kafka.Writer) {
	ctx := context.Background()
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading order message: %v", err)
			continue
		}

		var order OrderEvent
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			log.Printf("error decoding order message: %v", err)
			continue
		}

		remaining, ok := store.reserve(order.ProductID, order.Quantity)
		event := StockEvent{
			OrderID:      order.ID,
			ProductID:    order.ProductID,
			RemainingQty: remaining,
		}
		if ok {
			event.Status = "stock-updated"
			log.Printf("order %s: reserved %d x %s, %d remaining", order.ID, order.Quantity, order.ProductID, remaining)
		} else {
			event.Status = "out-of-stock"
			log.Printf("order %s: OUT OF STOCK for %s (requested %d, have %d)", order.ID, order.ProductID, order.Quantity, remaining)
		}

		payload, err := json.Marshal(event)
		if err != nil {
			log.Printf("error encoding stock event: %v", err)
			continue
		}

		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = writer.WriteMessages(writeCtx, kafka.Message{
			Key:   []byte(order.ProductID),
			Value: payload,
		})
		cancel()
		if err != nil {
			log.Printf("error publishing stock event: %v", err)
		}
	}
}

func handleStock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(store.snapshot())
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