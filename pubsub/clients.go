package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

type (
	// ClientConn represents a ClientConn connection to the core server
	ClientConn struct {
		conn     io.ReadWriteCloser // the connection to the core server
		encoder  *json.Encoder      //
		host     string             //
		messages chan Message       // the channel that core server 'publishes' updates to
		ctx      context.Context
	}
)

// New attempts to connect to a running core server at the clients specified
// host and port.
func NewClient(host string, ctx context.Context) (*ClientConn, error) {
	client := &ClientConn{
		host:     host,
		messages: make(chan Message),
		ctx:      ctx,
	}

	return client, client.connect()
}

// connect dials the remote core server and handles any incoming responses back
// from core
func (c *ClientConn) connect() error {

	// attempt to connect to the server
	conn, err := net.Dial("tcp", c.host)
	if err != nil {
		return fmt.Errorf("Failed to dial '%s' - %s", c.host, err.Error())
	}

	// set the connection for the client
	c.conn = conn

	// create a new json encoder for the clients connection
	c.encoder = json.NewEncoder(c.conn)

	// ensure we are authorized/still connected (unauthorized clients get disconnected)
	c.Ping()
	decoder := json.NewDecoder(conn)
	msg := Message{}
	if err := decoder.Decode(&msg); err != nil {
		conn.Close()
		close(c.messages)
		return fmt.Errorf("Ping failed, possibly bad token, or can't read from core - %s", err.Error())
	}

	// connection loop (blocking); continually read off the connection. Once something
	// is read, check to see if it's a message the client understands to be one of
	// its commands. If so attempt to execute the command.
	go func() {

	loop:
		for {
			msg := Message{}

			// decode an array value (Message)
			if err := decoder.Decode(&msg); err != nil {
				switch err {
				case io.EOF:
					logger.Debug("[pubsub client] pubsub terminated connection")
				case io.ErrUnexpectedEOF:
					logger.Debug("[pubsub client] pubsub terminated connection unexpectedly")
				default:
					logger.Debug("[pubsub client] Failed to get message from pubsub - %s", err.Error())
				}
				conn.Close()
				close(c.messages)
				return
			}
			c.messages <- msg // read from this using the .Messages() function
			logger.Debug("[pubsub client] Received message - %#v", msg)

			select {
			case <-c.ctx.Done():
				if conn != nil {
					conn.Close()
				}
				break loop
			default:

			}
		}
	}()

	return nil
}

// Ping the server
func (c *ClientConn) Ping() error {
	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "ping"})
	}
}

// Subscribe takes the specified tags and tells the server to subscribe to updates
// on those tags, returning the tags and an error or nil
func (c *ClientConn) Subscribe(tags []string) error {

	if len(tags) == 0 {
		return fmt.Errorf("Unable to subscribe - missing tags")
	}

	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "subscribe", Tags: tags})
	}
}

// Unsubscribe takes the specified tags and tells the server to unsubscribe from
// updates on those tags, returning an error or nil
func (c *ClientConn) Unsubscribe(tags []string) error {

	if len(tags) == 0 {
		return fmt.Errorf("Unable to unsubscribe - missing tags")
	}

	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "unsubscribe", Tags: tags})
	}
}

// Publish sends a message to the core server to be published to all subscribed
// clients
func (c *ClientConn) Publish(tags []string, data string) error {

	if len(tags) == 0 {
		return fmt.Errorf("Unable to publish - missing tags")
	}

	if data == "" {
		return fmt.Errorf("Unable to publish - missing data")
	}

	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "publish", Tags: tags, Data: data})
	}
}

// PublishAfter sends a message to the core server to be published to all subscribed
// clients after a specified delay
func (c *ClientConn) PublishAfter(tags []string, data string, delay time.Duration) error {
	go func() {
		<-time.After(delay)

		select {
		case <-c.ctx.Done():
			return
		default:
			c.Publish(tags, data)
		}
	}()
	return nil
}

// List requests a list from the server of the tags this client is subscribed to
func (c *ClientConn) List() error {
	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "list"})
	}
}

// listall related
// List requests a list from the server of the tags this client is subscribed to
func (c *ClientConn) ListAll() error {
	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "listall"})
	}
}

// who related
// Who requests connection/subscriber stats from the server
func (c *ClientConn) Who() error {
	select {
	case <-c.ctx.Done():
		return nil
	default:
		return c.encoder.Encode(&Message{Command: "who"})
	}
}

// Close closes the client data channel and the connection to the server
func (c *ClientConn) Close() {
	c.conn.Close()
	// close(c.messages) // we don't close this in case there is a message waiting in the channel
}

// Messages ...
func (c *ClientConn) Messages() <-chan Message {
	return c.messages
}
