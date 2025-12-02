package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) Publish(ctx context.Context, payload any, exchange, routingKey string, mandatory bool) error {
	c.mu.RLock()
	if !c.isReady {
		c.mu.RUnlock()
		return ErrNotConnected
	}
	channel := c.ch
	c.mu.RUnlock()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		mandatory,
		false, // immediate - deprecated
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
		},
	)
}
