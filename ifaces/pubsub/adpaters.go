package ifaces_pubsub

import (
	"context"
	"time"

	"cloud.google.com/go/pubsub"
)

// AdaptClient adapts a pubsub.Client so that it satisfies the Client
// interface.
func AdaptPubsubClient(c *pubsub.Client) PubsubClient {
	return pubsubClient{c}
}

// AdaptMessage adapts a pubsub.Message so that it satisfies the Message
// interface.
func AdaptPubsubMessage(msg *pubsub.Message) Message {
	return message{msg}
}

type (
	pubsubClient         struct{ *pubsub.Client }
	topic                struct{ *pubsub.Topic }
	subscription         struct{ *pubsub.Subscription }
	message              struct{ *pubsub.Message }
	publishResult        struct{ *pubsub.PublishResult }
	topicIterator        struct{ *pubsub.TopicIterator }
	subscriptionIterator struct{ *pubsub.SubscriptionIterator }
)

func (c pubsubClient) Topic(id string) Topic {
	return topic{c.Client.Topic(id)}
}

func (c pubsubClient) Topics(ctx context.Context) TopicIterator {
	return topicIterator{c.Client.Topics(ctx)}
}

func (c topicIterator) Next() (Topic, error) {
	t, err := c.TopicIterator.Next()
	return topic{t}, err
}

func (c subscriptionIterator) Next() (Subscription, error) {
	s, err := c.SubscriptionIterator.Next()
	return subscription{s}, err
}

// func (c client) CreateSubscription(ctx context.Context, id string, cfg SubscriptionConfig) (Subscription, error) {
// 	s, err := c.Client.CreateSubscription(ctx, id, cfg.toPS())
// 	if err != nil {
// 		return nil, err
// 	}
// 	return subscription{s}, nil
// }

func (t topic) String() string {
	return t.Topic.String()
}

func (t topic) Publish(ctx context.Context, msg Message) PublishResult {
	return publishResult{t.Topic.Publish(ctx, msg.(message).Message)}
}

func (t topic) Subscriptions(ctx context.Context) SubscriptionIterator {
	return subscriptionIterator{t.Topic.Subscriptions(ctx)}
}

func (t topic) Exists(ctx context.Context) (bool, error) {
	return t.Topic.Exists(ctx)
}

func (t topic) ID() string {
	return t.Topic.ID()
}

func (s subscription) ID() string {
	return s.Subscription.ID()
}

func (s subscription) String() string {
	return s.Subscription.String()
}

func (m message) ID() string {
	return m.Message.ID
}

func (m message) Data() []byte {
	return m.Message.Data
}

func (m message) Attributes() map[string]string {
	return m.Message.Attributes
}

func (m message) PublishTime() time.Time {
	return m.Message.PublishTime
}

func (r publishResult) Get(ctx context.Context) (serverID string, err error) {
	return r.PublishResult.Get(ctx)
}
