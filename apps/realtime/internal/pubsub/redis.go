package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Subscriber manages Redis Pub/Sub subscriptions for session rooms.
// It subscribes to channels when the first client joins a room
// and unsubscribes when the last client leaves.
type Subscriber struct {
	rdb     *redis.Client
	mu      sync.Mutex
	subs    map[string]*redis.PubSub // keyed by session code
	handler func(code string, message []byte)
}

// NewSubscriber creates a new Subscriber.
// handler is called with the session code and raw message bytes for each received message.
func NewSubscriber(rdb *redis.Client, handler func(code string, message []byte)) *Subscriber {
	return &Subscriber{
		rdb:     rdb,
		subs:    make(map[string]*redis.PubSub),
		handler: handler,
	}
}

// Subscribe subscribes to the Redis channel for a session code.
// Safe to call multiple times for the same code — only the first call creates a subscription.
func (s *Subscriber) Subscribe(code string) {
	s.mu.Lock()
	if _, ok := s.subs[code]; ok {
		s.mu.Unlock()
		return // already subscribed
	}

	channel := "session:" + code
	pubsub := s.rdb.Subscribe(context.Background(), channel)
	s.subs[code] = pubsub
	s.mu.Unlock()

	slog.Info("subscribed to redis channel", "channel", channel)

	// Listen for messages in a goroutine
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			slog.Debug("received redis message", "channel", msg.Channel, "size", len(msg.Payload))
			s.handler(code, []byte(msg.Payload))
		}
		slog.Info("redis channel closed", "channel", channel)
	}()
}

// Unsubscribe unsubscribes from the Redis channel for a session code.
func (s *Subscriber) Unsubscribe(code string) {
	s.mu.Lock()
	pubsub, ok := s.subs[code]
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(s.subs, code)
	s.mu.Unlock()

	channel := "session:" + code
	if err := pubsub.Close(); err != nil {
		slog.Error("failed to close redis subscription", "channel", channel, "error", err)
	} else {
		slog.Info("unsubscribed from redis channel", "channel", channel)
	}
}

// Close closes all active subscriptions.
func (s *Subscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for code, pubsub := range s.subs {
		if err := pubsub.Close(); err != nil {
			slog.Error("failed to close subscription", "code", code, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	s.subs = make(map[string]*redis.PubSub)

	if firstErr != nil {
		return fmt.Errorf("close subscriptions: %w", firstErr)
	}
	return nil
}

// NewRedisClient creates a Redis client from a URL.
func NewRedisClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	rdb := redis.NewClient(opts)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("unable to ping redis: %w", err)
	}

	return rdb, nil
}
