package rabbitmq

import (
	"context"
	"sync"
	"time"

	retry "github.com/avast/retry-go/v5"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type HandlerFunc func(ctx context.Context, body []byte) error

func (c *Client) StartConsumer(ctx context.Context, config Consumer, handler HandlerFunc) error {
	if config.RetryMax == 0 {
		config.RetryMax = 3
	}
	if config.RetryStart == 0 {
		config.RetryStart = 1 * time.Second
	}
	if config.Workers == 0 {
		config.Workers = 1
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.mu.RLock()
		ready := c.isReady
		channel := c.ch
		c.mu.RUnlock()

		if !ready {
			time.Sleep(time.Second)
			continue
		}

		if err := channel.Qos(config.PrefetchCount, 0, false); err != nil {
			logrus.Errorf("RabbitMQ: Failed to set QoS: %v", err)
			time.Sleep(time.Second)
			continue
		}

		msgs, err := channel.Consume(
			config.Queue,
			config.Name,
			false, // AutoAck = false (Manual Ack is safer)
			false, // Exclusive
			false, // NoLocal
			false, // NoWait
			nil,   // Args
		)

		if err != nil {
			logrus.Errorf("RabbitMQ: Consume failed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		c.consumeLoop(ctx, msgs, config, handler)
	}
}

func (c *Client) consumeLoop(ctx context.Context, msgs <-chan amqp.Delivery, config Consumer, handler HandlerFunc) {
	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgs:
					if !ok {
						return
					}
					c.processMessage(ctx, msg, config, handler)
				}
			}
		}()
	}
	wg.Wait()
}

func (c *Client) processMessage(ctx context.Context, msg amqp.Delivery, config Consumer, handler HandlerFunc) {
	err := retry.New(
		retry.Attempts(uint(config.RetryMax+1)),
		retry.Delay(config.RetryStart),
		retry.DelayType(retry.BackOffDelay),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			logrus.Errorf("RabbitMQ: Retry %d failed: %v", n, err)
		})).Do((func() error {
		return handler(ctx, msg.Body)
	}))

	if err == nil {
		if ackErr := msg.Ack(false); ackErr != nil {
			logrus.Errorf("RabbitMQ: Failed to ack message: %v", ackErr)
		}
		return
	}

	logrus.Errorf("RabbitMQ: Message failed after retries. Error: %v. Sending to DLQ (Reject).", err)

	// Reject(false) sends the message to the Dead Letter Exchange (if configured on the queue)
	// or discards it if no DLQ is configured.
	if nackErr := msg.Reject(false); nackErr != nil {
		logrus.Errorf("RabbitMQ: Failed to nack message: %v", nackErr)
	}
}
