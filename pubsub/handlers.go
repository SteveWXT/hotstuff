package pubsub

import (
	"fmt"
	"strings"
)

// GenerateHandlers ...
func GenerateHandlers() map[string]HandleFunc {
	return map[string]HandleFunc{
		"ping":        handlePing,
		"subscribe":   handleSubscribe,
		"unsubscribe": handleUnsubscribe,
		"publish":     handlePublish,
		// "publishAfter":     handlePublishAfter,
		"list":    handleList,
		"listall": handleListAll, // listall related
		"who":     handleWho,     // who related
	}
}

// handlePing
func handlePing(proxy *Proxy, msg Message) error {
	// goroutining any of these would allow a client to spam and overwhelm the server. clients don't need the ability to ping indefinitely
	proxy.Pipe <- Message{Command: "ping", Tags: []string{}, Data: "pong"}
	return nil
}

// handleSubscribe
func handleSubscribe(proxy *Proxy, msg Message) error {
	proxy.Subscribe(msg.Tags)
	return nil
}

// handleUnsubscribe
func handleUnsubscribe(proxy *Proxy, msg Message) error {
	proxy.Unsubscribe(msg.Tags)
	return nil
}

// handlePublish
func handlePublish(proxy *Proxy, msg Message) error {
	proxy.Publish(msg.Tags, msg.Data)
	return nil
}

// handleList
func handleList(proxy *Proxy, msg Message) error {
	var subscriptions string
	for _, v := range proxy.List() {
		subscriptions += strings.Join(v, ",")
	}
	proxy.Pipe <- Message{Command: "list", Tags: msg.Tags, Data: subscriptions}
	return nil
}

// handleListAll - listall related
func handleListAll(proxy *Proxy, msg Message) error {
	subscriptions := Subscribers()
	proxy.Pipe <- Message{Command: "listall", Tags: msg.Tags, Data: subscriptions}
	return nil
}

// handleWho - who related
func handleWho(proxy *Proxy, msg Message) error {
	who, max := Who()
	subscribers := fmt.Sprintf("Lifetime  connections: %d\nSubscribers connected: %d", max, who)
	proxy.Pipe <- Message{Command: "who", Tags: msg.Tags, Data: subscribers}
	return nil
}
