package interfaces

import (
	"context"
	"l0/internal/models"
	"time"
)

type DeadLetterMessage struct {
	ID            string    `json:"id"`
	OriginalTopic string    `json:"original_topic"`
	Partition     int       `json:"partition"`
	Offset        int64     `json:"offset"`
	Message       []byte    `json:"message"`
	Reason        string    `json:"reason"`
	Error         string    `json:"error"`
	Timestamp     time.Time `json:"timestamp"`
	RetryCount    int       `json:"retry_count"`
}

type DeadLetterQueue interface {
	Send(message []byte, topic string, partition int, offset int64, reason string, originalError error) error
	Get(limit int) ([]DeadLetterMessage, error)
	Retry(messageID string) error
}

type OrderProcessor interface {
	ProcessOrder(ctx context.Context, order *models.Order) error
}