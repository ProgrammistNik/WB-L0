package kafka

import (
	"fmt"
	"github.com/rs/zerolog"
	"l0/internal/interfaces"
	"sync"
	"time"
)

// An InMemoryDeadLetterQueue provides a simple in-memory implementation of dead letter queue
type InMemoryDeadLetterQueue struct {
	mu        sync.RWMutex
	messages  map[string]*interfaces.DeadLetterMessage
	logger    *zerolog.Logger
	idCounter int64
}

// NewInMemoryDeadLetterQueue creates a new instance of in-memory dead letter queue
func NewInMemoryDeadLetterQueue(logger *zerolog.Logger) *InMemoryDeadLetterQueue {
	return &InMemoryDeadLetterQueue{
		messages: make(map[string]*interfaces.DeadLetterMessage),
		logger:   logger,
	}
}

// Send adds a message with additional information to the dead letter queue
func (dlq *InMemoryDeadLetterQueue) Send(
	message []byte, topic string, partition int, offset int64, reason string,
	originalError error,
) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	dlq.idCounter++
	messageID := fmt.Sprintf("dlq_%d_%d", time.Now().Unix(), dlq.idCounter)

	errorMsg := ""
	if originalError != nil {
		errorMsg = originalError.Error()
	}

	dlqMessage := &interfaces.DeadLetterMessage{
		ID:            messageID,
		OriginalTopic: topic,
		Partition:     partition,
		Offset:        offset,
		Message:       make([]byte, len(message)),
		Reason:        reason,
		Error:         errorMsg,
		Timestamp:     time.Now(),
		RetryCount:    0,
	}

	copy(dlqMessage.Message, message)

	dlq.messages[messageID] = dlqMessage

	dlq.logger.Error().
		Str("message_id", messageID).
		Str("topic", topic).
		Int("partition", partition).
		Int64("offset", offset).
		Str("reason", reason).
		Str("error", errorMsg).
		Int("message_size", len(message)).
		Msg("Message sent to dead letter queue")

	return nil
}

// Get returns not more than limit messages from dead letter queue
func (dlq *InMemoryDeadLetterQueue) Get(limit int) ([]interfaces.DeadLetterMessage, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	messages := make([]interfaces.DeadLetterMessage, 0, limit)
	count := 0

	for _, msg := range dlq.messages {
		if count >= limit {
			break
		}
		msgCopy := *msg
		msgCopy.Message = make([]byte, len(msg.Message))
		copy(msgCopy.Message, msg.Message)

		messages = append(messages, msgCopy)
		count++
	}

	return messages, nil
}

func (dlq *InMemoryDeadLetterQueue) Retry(messageID string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	message, ok := dlq.messages[messageID]

	if !ok {
		return fmt.Errorf("dead letter message with ID %s not found", messageID)
	}

	message.RetryCount++

	return nil
}

// GetMessageCount returns the total number of messages in the dead letter queue
func (dlq *InMemoryDeadLetterQueue) GetMessageCount() int {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	return len(dlq.messages)
}

// Clear removes all the messages from the dead letter queue
func (dlq *InMemoryDeadLetterQueue) Clear() {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	dlq.messages = make(map[string]*interfaces.DeadLetterMessage)
}

// GetByReason returns not more than limit messages that have specified reason
func (dlq *InMemoryDeadLetterQueue) GetByReason(reason string, limit int) ([]interfaces.DeadLetterMessage, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	messages := make([]interfaces.DeadLetterMessage, 0, limit)
	count := 0

	for _, msg := range dlq.messages {
		if count >= limit {
			break
		}

		if msg.Reason == reason {
			msgCopy := *msg
			msgCopy.Message = make([]byte, len(msg.Message))
			copy(msgCopy.Message, msg.Message)

			messages = append(messages, msgCopy)
			count++
		}
	}
	return messages, nil
}

// Statistics returns the statistics for the dead letter queue
func (dlq *InMemoryDeadLetterQueue) Statistics() map[string]any {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	stats := make(map[string]any)
	reasonCounts := make(map[string]int)
	topicCounts := make(map[string]int)

	for _, msg := range dlq.messages {
		reasonCounts[msg.Reason]++
		topicCounts[msg.Reason]++
	}

	stats["total_messages"] = len(dlq.messages)
	stats["messages_by_reason"] = reasonCounts
	stats["messages_by_topic"] = topicCounts

	return stats
}