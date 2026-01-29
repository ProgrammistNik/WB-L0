package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"github.com/sony/gobreaker"
	"l0/internal/config"
	"l0/internal/interfaces"
	"l0/internal/models"
	"strings"
	"sync"
	"time"
)

type Consumer struct {
	reader          *kafka.Reader
	config          config.KafkaConfig
	mu              sync.RWMutex
	running         bool
	processor       interfaces.OrderProcessor
	logger          *zerolog.Logger
	circuitBreaker  *gobreaker.CircuitBreaker
	deadLetterQueue interfaces.DeadLetterQueue
	brokers         []string
}

func NewConsumer(config config.Config, processor interfaces.OrderProcessor, logger *zerolog.Logger) *Consumer {
	deadLetterQueue := NewInMemoryDeadLetterQueue(logger)
	cb := gobreaker.NewCircuitBreaker(
		gobreaker.Settings{
			Name:        "kafka-consumer",
			MaxRequests: uint32(config.CircuitBreaker.HalfOpenMaxCalls),
			Interval:    config.CircuitBreaker.Timeout,
			Timeout:     config.CircuitBreaker.Timeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= uint32(config.CircuitBreaker.MaxFailers)
			},
		},
	)

	return &Consumer{
		config:          config.Kafka,
		processor:       processor,
		logger:          logger,
		circuitBreaker:  cb,
		deadLetterQueue: deadLetterQueue,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("consumer is already running")
	}

	c.brokers = strings.Split(c.config.Listeners, ",")
	for i, broker := range c.brokers {
		c.brokers[i] = strings.TrimSpace(broker)
	}

	c.reader = kafka.NewReader(
		kafka.ReaderConfig{
			Brokers:     c.brokers,
			Topic:       c.config.Topic,
			GroupID:     c.config.GroupID,
			StartOffset: kafka.LastOffset,
			MinBytes:    10e3,
			MaxBytes:    10e6,
			MaxWait:     time.Second,
			ErrorLogger: kafka.LoggerFunc(
				func(msg string, args ...interface{}) {
					c.logger.Error().
						Str("kafka_error", fmt.Sprintf(msg, args...)).
						Msg("kafka reader error")

				},
			),
		},
	)

	if strings.TrimSpace(c.config.GroupID) == "" {
		c.logger.Warn().Msg("Kafka GroupID is empty — offsets will NOT be committed. Set GroupID to enable consumer-group offset commits.")
	}

	c.running = true

	go c.consume(ctx)

	return nil
}

func (c *Consumer) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false

	if c.reader != nil {
		if err := c.reader.Close(); err != nil {
			c.logger.Error().Err(err).Msg("Error closing Kafka reader")
			return fmt.Errorf("failed to close Kafka reader: %w", err)
		}
		c.reader = nil
	}

	return nil
}

func (c *Consumer) consume(ctx context.Context) {
	for {
		c.mu.RLock()
		running := c.running
		reader := c.reader
		c.mu.RUnlock()

		if !running || reader == nil {
			break
		}

		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := c.circuitBreaker.Execute(
			func() (any, error) {
				defer cancel()
				return reader.FetchMessage(fetchCtx)
			},
		)

		if err != nil {
			if ctx.Err() != nil {
				break
			}
			c.logger.Error().Err(err).Msg("Error fetching Kafka message")

			retryErr := retry.Do(
				func() error {
					return nil
				},
				retry.Attempts(3),
				retry.Delay(1*time.Second),
				retry.DelayType(retry.BackOffDelay),
				retry.MaxDelay(30*time.Second),
				retry.OnRetry(
					func(n uint, err error) {
						c.logger.Warn().
							Uint("attempt", n+1).
							Msg("Retrying Kafka connection")
					},
				),
				retry.Context(ctx),
			)

			if retryErr != nil {
				c.logger.Error().Err(retryErr).Msg("Failed to recover Kafka connection")
				break
			}
			continue
		}
		message := result.(kafka.Message)

		processErr := c.processMessage(ctx, message)
		if processErr != nil {
			c.logger.Error().
				Err(processErr).
				Str("topic", message.Topic).
				Int("partition", message.Partition).
				Int64("offset", message.Offset).
				Msg("Error processing message, sending to dead letter queue")

			dlqErr := c.deadLetterQueue.Send(
				message.Value,
				message.Topic,
				message.Partition,
				message.Offset,
				"processing_error",
				processErr,
			)
			if dlqErr != nil {
				c.logger.Error().
					Err(dlqErr).
					Str("topic", message.Topic).
					Int("partition", message.Partition).
					Int64("offset", message.Offset).
					Msg("Failed to send message to dead letter queue")
			}
		}

		if strings.TrimSpace(c.config.GroupID) != "" {
			commitErr := retry.Do(
				func() error {
					return reader.CommitMessages(ctx, message)
				},
				retry.Attempts(5),
				retry.Delay(500*time.Millisecond),
				retry.DelayType(retry.BackOffDelay),
				retry.Context(ctx),
			)

			if commitErr != nil {
				c.logger.Error().
					Err(commitErr).
					Str("topic", message.Topic).
					Int("partition", message.Partition).
					Int64("offset", message.Offset).
					Msg("Failed to commit message after retries")
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, message kafka.Message) error {
	start := time.Now()

	var order models.Order
	if err := json.Unmarshal(message.Value, &order); err != nil {
		c.logger.Error().
			Err(err).
			Str("topic", message.Topic).
			Int64("offset", message.Offset).
			Str("raw_message", string(message.Value)).
			Msg("Failed to unmarshal order JSON")

		dlqErr := c.deadLetterQueue.Send(
			message.Value,
			message.Topic,
			message.Partition,
			message.Offset,
			"json_unmarshal_error",
			err,
		)
		if dlqErr != nil {
			c.logger.Error().Err(dlqErr).Msg("Failed to send malformed JSON to dead letter queue")
		}

		return fmt.Errorf("failed to unmarshal order JSON: %w", err)
	}

	if err := c.validateOrder(&order); err != nil {
		c.logger.Error().
			Err(err).
			Str("order_uid", order.OrderUID).
			Str("topic", message.Topic).
			Int64("offset", message.Offset).
			Msg("Order validation failed")

		dlqErr := c.deadLetterQueue.Send(
			message.Value,
			message.Topic,
			message.Partition,
			message.Offset,
			"validation_error",
			err,
		)

		if dlqErr != nil {
			c.logger.Error().Err(dlqErr).Msg("Failed to send invalid order to dead letter queue")
		}

		return fmt.Errorf("order validation failed: %w", err)
	}

	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := c.processor.ProcessOrder(processCtx, &order); err != nil {
		c.logger.Error().
			Err(err).
			Str("order_uid", order.OrderUID).
			Str("topic", message.Topic).
			Int64("offset", message.Offset).
			Dur("duration", time.Since(start)).
			Msg("Failed to process order")

		return fmt.Errorf("failed to process order: %w", err)
	}

	return nil
}

func (c *Consumer) validateOrder(order *models.Order) error {
	if err := order.Validate(); err != nil {
		c.logger.Error().
			Err(err).
			Str("order_uid", order.OrderUID).
			Msg("Kafka consumer: order validation failed")
		return err
	}
	return nil
}

func (c *Consumer) GetDeadLetterQueue() interfaces.DeadLetterQueue {
	return c.deadLetterQueue
}