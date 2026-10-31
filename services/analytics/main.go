package main

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// OrderEvent mirrors what the order-service publishes.
type OrderEvent struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Customer  string    `json:"customer"`
	CreatedAt time.Time `json:"created_at"`
}

// StockEvent mirrors what the inventory-service publishes.
type StockEvent struct {
	OrderID      string `json:"order_id"`
	ProductID    string `json:"product_id"`
	Status       string `json:"status"` // "stock-updated" or "out-of-stock"
	RemainingQty int    `json:"remaining_qty"`
}

// Stats is what GET /stats returns.
type Stats struct {
	TotalOrders      int            `json:"total_orders"`
	UnitsOrdered     int            `json:"units_ordered"`
	ConfirmedOrders  int            `json:"confirmed_orders"`
	OutOfStockOrders int            `json:"out_of_stock_orders"`
	OrdersByProduct  map[string]int `json:"orders_by_product"`
	OrdersByDay      map[string]int `json:"orders_by_day"`
}

// statsStore holds the running totals. Several goroutines update it at the
// same time (one per partition), so a mutex allows only one change at a time.
type statsStore struct {
	mu    sync.Mutex
	stats Stats
}

func newStatsStore() *statsStore {
	return &statsStore{
		stats: Stats{
			OrdersByProduct: map[string]int{},
			OrdersByDay:     map[string]int{},
		},
	}
}

// recordOrder handles one message from the order-created topic.
func (s *statsStore) recordOrder(value []byte) {
	var order OrderEvent
	if err := json.Unmarshal(value, &order); err != nil {
		log.Printf("skipping bad order message: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.TotalOrders++
	s.stats.UnitsOrdered += order.Quantity
	s.stats.OrdersByProduct[order.ProductID]++
	// Go formats dates from an example date: "2006-01-02" means YYYY-MM-DD (UTC here).
	s.stats.OrdersByDay[order.CreatedAt.Format("2006-01-02")]++
}

// recordStock handles one message from the stock-updated topic.
func (s *statsStore) recordStock(value []byte) {
	var event StockEvent
	if err := json.Unmarshal(value, &event); err != nil {
		log.Printf("skipping bad stock message: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Status {
	case "stock-updated":
		s.stats.ConfirmedOrders++
	case "out-of-stock":
		s.stats.OutOfStockOrders++
	}
}

// snapshot returns a separate copy of the stats, so the HTTP handler can
// convert it to JSON while the goroutines continue to update the original.
func (s *statsStore) snapshot() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.stats
	out.OrdersByProduct = maps.Clone(s.stats.OrdersByProduct)
	out.OrdersByDay = maps.Clone(s.stats.OrdersByDay)
	return out
}

var store = newStatsStore()

func main() {
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	orderTopic := getEnv("ORDER_TOPIC", "order-created")
	stockTopic := getEnv("STOCK_TOPIC", "stock-updated")
	port := getEnv("PORT", "8084")

	// Read both topics in the background. Each call receives the function
	// that handles one message (in Go, a function can be passed as a value).
	go readFromStart(kafkaBroker, orderTopic, store.recordOrder)
	go readFromStart(kafkaBroker, stockTopic, store.recordStock)

	mux := http.NewServeMux()
	mux.HandleFunc("/stats", handleStats)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("analytics-service listening on :%s (reading %s and %s)", port, orderTopic, stockTopic)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// readFromStart reads every partition of a topic from the oldest stored
// message, then keeps waiting for new messages.
//
// Unlike inventory and notification, this service has no GroupID (no consumer
// group). Kafka therefore stores no committed offsets for it: every time the
// service starts, it reads all messages again and rebuilds its totals.
func readFromStart(broker, topic string, handle func([]byte)) {
	for _, partition := range partitionsOf(broker, topic) {
		// Kafka assigns partitions only to consumer groups,
		// so this service opens one reader per partition itself.
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:   []string{broker},
			Topic:     topic,
			Partition: partition,
			MinBytes:  1,
			MaxBytes:  10e6,
		})
		if err := reader.SetOffset(kafka.FirstOffset); err != nil { // start at the oldest stored message
			log.Printf("could not rewind %s partition %d: %v", topic, partition, err)
		}
		log.Printf("reading %s partition %d from the start", topic, partition)

		go func() {
			for {
				msg, err := reader.ReadMessage(context.Background())
				if err != nil {
					log.Printf("error reading %s: %v", topic, err)
					time.Sleep(time.Second) // wait a bit before trying again
					continue
				}
				handle(msg.Value)
			}
		}()
	}
}

// partitionsOf asks Kafka which partitions a topic has.
// If the topic doesn't exist yet, it waits and asks again.
func partitionsOf(broker, topic string) []int {
	for {
		conn, err := kafka.Dial("tcp", broker)
		if err == nil {
			parts, readErr := conn.ReadPartitions(topic)
			conn.Close()
			if readErr == nil && len(parts) > 0 {
				ids := make([]int, len(parts))
				for i, p := range parts {
					ids[i] = p.ID
				}
				return ids
			}
		}
		log.Printf("topic %s not ready yet, retrying in 2s...", topic)
		time.Sleep(2 * time.Second)
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
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