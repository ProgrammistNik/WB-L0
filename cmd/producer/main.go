package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"

	"math/rand"
	"time"

	"l0/internal/models"
)

func generateOrder() *models.Order {
	now := time.Now()
	id := rand.Intn(100000000)

	return &models.Order{
		OrderUID:    fmt.Sprintf("test%08d", id),
		TrackNumber: fmt.Sprintf("TRACK%08d", id),
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Test User",
			Phone:   "+7(999)777-32-32",
			Zip:     "12345",
			City:    "Test City",
			Address: "123 Test St",
			Region:  "NY",
			Email:   "test@example.com",
		},
		Payment: models.Payment{
			Transaction:  fmt.Sprintf("test%08d", id),
			RequestID:    fmt.Sprintf("req%08d", id),
			Currency:     "USD",
			Provider:     "test",
			Amount:       1000,
			PaymentDt:    now.Unix(),
			Bank:         "test",
			DeliveryCost: 100,
			GoodsTotal:   900,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      int64(id),
				TrackNumber: fmt.Sprintf("TRACK%08d", id),
				Price:       1000,
				Rid:         fmt.Sprintf("rid%08d", id),
				Name:        "Test Item",
				Sale:        10,
				Size:        "M",
				TotalPrice:  900,
				NmID:        int64(id),
				Brand:       "Test",
				Status:      200,
			},
		},
		Locale:            "en",
		InternalSignature: fmt.Sprintf("sig%08d", id),
		CustomerID:        fmt.Sprintf("cust%08d", id),
		DeliveryService:   "test",
		Shardkey:          "1",
		SmID:              1,
		DateCreated:       now,
		OofShard:          "1",
	}
}

func main() {
	godotenv.Load("deployments/.env")

	count := flag.Int("count", 1, "Number of orders")
	flag.Parse()

	var brokers string
	if env := os.Getenv("KAFKA_BROKERS"); env != "" {
		brokers = env
	}
	var topic string
	if env := os.Getenv("KAFKA_TOPIC"); env != "" {
		topic = env
	}

	writer := &kafka.Writer{
		Addr:  kafka.TCP(strings.Split(brokers, ",")...),
		Topic: topic,
	}
	defer func(writer *kafka.Writer) {
		err := writer.Close()
		if err != nil {
			log.Printf("Error in closing writer")
		}
	}(writer)

	ctx := context.Background()
	for i := range *count {
		order := generateOrder()
		data, _ := json.Marshal(order)

		err := writer.WriteMessages(
			ctx, kafka.Message{
				Key:   []byte(order.OrderUID),
				Value: data,
			},
		)

		if err != nil {
			log.Printf("Failed to send order %d: %v", i+1, err)
		} else {
			fmt.Printf("Sent order: %s\n", order.OrderUID)
		}
	}
}