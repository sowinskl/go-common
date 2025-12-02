package rabbitmq

import (
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// Client manages the RabbitMQ connection and channel
type Client struct {
	cfg    Config
	conn   *amqp.Connection
	ch     *amqp.Channel
	mu     sync.RWMutex // Protects access to conn, ch, and isReady
	closed bool         // Indicates if the client was explicitly closed

	// Channels for notifying when the connection/channel drops
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error

	// isReady is true only when both Connection and Channel are established
	isReady bool
}

// NewClient creates a new Client.
// It attempts to connect immediately. If the initial connection fails, it returns an error (Fail Fast).
// If successful, it launches a background goroutine to handle future reconnections indefinitely.
func NewClient(cfg Config) (*Client, error) {
	if cfg.ReconnectDelay == 0 {
		cfg.ReconnectDelay = time.Second
	}

	client := &Client{
		cfg: cfg,
	}

	// Attempt initial connection synchronously.
	// If this fails, the app should probably exit (fail fast).
	if err := client.connect(); err != nil {
		return nil, err
	}

	// Start the reconnection manager
	go client.handleReconnection()

	return client, nil
}

// Close shuts down the connection cleanly.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// handleReconnection monitors the connection state and attempts to reconnect when lost.
// It will retry indefinitely until successful or the client is closed.
func (c *Client) handleReconnection() {
	for {
		c.mu.RLock()
		closed := c.closed
		c.mu.RUnlock()

		if closed {
			return
		}

		// connected - wait for a close signal
		select {
		case err := <-c.notifyConnClose:
			logrus.Printf("RabbitMQ: Connection lost: %v", err)
		case err := <-c.notifyChanClose:
			logrus.Printf("RabbitMQ: Channel lost: %v", err)
		}

		// Connection is down. Mark as not ready
		c.setReady(false)

		for {
			c.mu.RLock()
			closed := c.closed
			c.mu.RUnlock()

			if closed {
				return
			}

			logrus.Printf("RabbitMQ: Attempting to reconnect...")

			if err := c.connect(); err != nil {
				log.Printf("RabbitMQ: Reconnection failed: %v. Retrying in %v...", err, c.cfg.ReconnectDelay)
				time.Sleep(c.cfg.ReconnectDelay)
				continue
			}

			// Reconnected successfully!
			// Break the inner loop to go back to waiting for close signals.
			log.Println("RabbitMQ: Reconnected!")
			break
		}
	}
}

func (c *Client) connect() error {
	conn, err := amqp.DialConfig(c.cfg.URL, amqp.Config{
		Properties: amqp.Table{"connection_name": c.cfg.AppName},
	})
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn = conn
	c.ch = ch
	c.notifyChanClose = make(chan *amqp.Error)
	c.notifyConnClose = make(chan *amqp.Error)
	c.ch.NotifyClose(c.notifyChanClose)
	c.conn.NotifyClose(c.notifyConnClose)
	c.isReady = true
	return nil
}

func (c *Client) setReady(ready bool) {
	c.mu.Lock()
	c.isReady = ready
	c.mu.Unlock()
}
