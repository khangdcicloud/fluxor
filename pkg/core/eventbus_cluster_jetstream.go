package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/nats-io/nats.go"
)

// ClusterJetStreamConfig configures the JetStream-backed (durable) cluster EventBus.
//
// Semantics:
// - Publish: delivered to each SERVICE once (load-balanced among replicas of the same service)
// - Send: work-queue semantics (delivered once to one replica)
// - Request: uses core NATS request/reply (not JetStream) for low latency
type ClusterJetStreamConfig struct {
	// URL is the NATS server URL, e.g. "nats://127.0.0.1:4222".
	URL string

	// Prefix is prepended to all subjects. Default: "fluxor".
	Prefix string

	// Service identifies the current service (e.g., "api-gateway", "payment-service").
	// Required for Publish fanout semantics (each service gets a copy).
	Service string

	// Name is an optional NATS connection name.
	Name string

	// RequestTimeout is the default timeout used by Request when timeout==0.
	RequestTimeout time.Duration

	// StreamMaxAge configures how long published messages are retained in JetStream streams.
	// Default: 10m.
	StreamMaxAge time.Duration

	// StreamStorage configures the backing store for streams. Default: nats.MemoryStorage.
	StreamStorage nats.StorageType

	// StreamReplicas configures stream replication factor. Default: 1.
	StreamReplicas int

	// AckWait is how long JetStream waits for an ACK before re-delivering.
	// Default: 30s.
	AckWait time.Duration

	// MaxAckPending bounds in-flight, unacked messages per consumer. Default: 1024.
	MaxAckPending int

	// ExecutorConfig controls bounded handler execution.
	// If zero, defaults are used.
	ExecutorConfig concurrency.ExecutorConfig
}

// NewClusterEventBusJetStream creates a clustered EventBus backed by NATS JetStream for durability.
func NewClusterEventBusJetStream(ctx context.Context, vertx Vertx, cfg ClusterJetStreamConfig) (EventBus, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx cannot be nil")
	}
	if vertx == nil {
		return nil, fmt.Errorf("vertx cannot be nil")
	}

	url := cfg.URL
	if url == "" {
		url = nats.DefaultURL
	}
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "fluxor"
	}
	if strings.TrimSpace(cfg.Service) == "" {
		return nil, fmt.Errorf("service is required for JetStream EventBus")
	}
	reqTimeout := cfg.RequestTimeout
	if reqTimeout <= 0 {
		reqTimeout = 5 * time.Second
	}

	maxAge := cfg.StreamMaxAge
	if maxAge <= 0 {
		maxAge = 10 * time.Minute
	}
	storage := cfg.StreamStorage
	if storage == 0 {
		storage = nats.MemoryStorage
	}
	replicas := cfg.StreamReplicas
	if replicas <= 0 {
		replicas = 1
	}
	ackWait := cfg.AckWait
	if ackWait <= 0 {
		ackWait = 30 * time.Second
	}
	maxAckPending := cfg.MaxAckPending
	if maxAckPending <= 0 {
		maxAckPending = 1024
	}

	execCfg := cfg.ExecutorConfig
	if execCfg.Workers == 0 && execCfg.QueueSize == 0 {
		execCfg = concurrency.DefaultExecutorConfig()
		execCfg.Workers = 16
		execCfg.QueueSize = 4096
	}

	nc, err := nats.Connect(url, func(o *nats.Options) error {
		if cfg.Name != "" {
			o.Name = cfg.Name
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	eb := &clusterJSEventBus{
		ctx:            ctx,
		vertx:          vertx,
		nc:             nc,
		js:             js,
		prefix:         prefix,
		service:        cfg.Service,
		requestTimeout: reqTimeout,
		ackWait:        ackWait,
		maxAckPending:  maxAckPending,
		executor:       concurrency.NewExecutor(ctx, execCfg),
		logger:         NewDefaultLogger(),
	}

	// Ensure streams exist (idempotent).
	if err := eb.ensureStreams(maxAge, storage, replicas); err != nil {
		_ = eb.Close()
		return nil, err
	}

	return eb, nil
}

type clusterJSEventBus struct {
	ctx   context.Context
	vertx Vertx

	nc *nats.Conn
	js nats.JetStreamContext

	prefix  string
	service string

	requestTimeout time.Duration

	ackWait       time.Duration
	maxAckPending int

	executor concurrency.Executor
	logger   Logger

	mu        sync.Mutex
	consumers []*clusterJSConsumer
}

func (eb *clusterJSEventBus) ensureStreams(maxAge time.Duration, storage nats.StorageType, replicas int) error {
	pubStream := eb.streamPub()
	sendStream := eb.streamSend()

	// Publish stream retains events for fanout to services.
	if _, err := eb.js.StreamInfo(pubStream); err != nil {
		if _, err := eb.js.AddStream(&nats.StreamConfig{
			Name:      pubStream,
			Subjects:  []string{eb.prefix + ".pub.>"},
			Storage:   storage,
			MaxAge:    maxAge,
			Retention: nats.LimitsPolicy,
			Replicas:  replicas,
		}); err != nil {
			return err
		}
	}

	// Send stream retains work-queue messages.
	if _, err := eb.js.StreamInfo(sendStream); err != nil {
		if _, err := eb.js.AddStream(&nats.StreamConfig{
			Name:      sendStream,
			Subjects:  []string{eb.prefix + ".send.>"},
			Storage:   storage,
			MaxAge:    maxAge,
			Retention: nats.LimitsPolicy,
			Replicas:  replicas,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (eb *clusterJSEventBus) Publish(address string, body interface{}) error {
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}

	data, err := encodeBody(body)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: eb.subjectPub(address),
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := GetRequestID(eb.ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	_, err = eb.js.PublishMsg(msg)
	return err
}

func (eb *clusterJSEventBus) Send(address string, body interface{}) error {
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}

	data, err := encodeBody(body)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: eb.subjectSend(address),
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := GetRequestID(eb.ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	_, err = eb.js.PublishMsg(msg)
	return err
}

func (eb *clusterJSEventBus) Request(address string, body interface{}, timeout time.Duration) (Message, error) {
	// Keep Request/Reply as core NATS for low-latency synchronous calls.
	if err := ValidateAddress(address); err != nil {
		return nil, err
	}
	if err := ValidateBody(body); err != nil {
		return nil, err
	}
	if err := ValidateTimeout(timeout); err != nil {
		return nil, err
	}

	data, err := encodeBody(body)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = eb.requestTimeout
	}

	msg := &nats.Msg{
		Subject: eb.subjectReq(address),
		Data:    data,
		Header:  nats.Header{},
	}
	if rid := GetRequestID(eb.ctx); rid != "" {
		msg.Header.Set("X-Request-ID", rid)
	}

	resp, err := eb.nc.RequestMsg(msg, timeout)
	if err != nil {
		return nil, err
	}

	h := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			h[k] = v[0]
		}
	}

	return &clusterNATSMessage{
		body:         resp.Data,
		headers:      h,
		replySubject: "",
		eb: &clusterNATSEventBus{
			ctx:            eb.ctx,
			vertx:          eb.vertx,
			nc:             eb.nc,
			prefix:         eb.prefix,
			requestTimeout: eb.requestTimeout,
			executor:       eb.executor,
			logger:         eb.logger,
		},
	}, nil
}

func (eb *clusterJSEventBus) Consumer(address string) Consumer {
	return newClusterJSConsumer(address, eb)
}

func (eb *clusterJSEventBus) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eb.mu.Lock()
	cons := eb.consumers
	eb.consumers = nil
	eb.mu.Unlock()

	for _, c := range cons {
		_ = c.Unregister()
	}

	_ = eb.executor.Shutdown(ctx)
	_ = eb.nc.Drain()
	eb.nc.Close()
	return nil
}

func (eb *clusterJSEventBus) streamPub() string  { return sanitizeStreamName(eb.prefix) + "_PUB" }
func (eb *clusterJSEventBus) streamSend() string { return sanitizeStreamName(eb.prefix) + "_SEND" }

func (eb *clusterJSEventBus) subjectPub(address string) string { return eb.prefix + ".pub." + address }
func (eb *clusterJSEventBus) subjectSend(address string) string {
	return eb.prefix + ".send." + address
}
func (eb *clusterJSEventBus) subjectReq(address string) string { return eb.prefix + ".req." + address }

type clusterJSConsumer struct {
	address string
	eb      *clusterJSEventBus

	mu         sync.Mutex
	handler    MessageHandler
	subs       []*nats.Subscription
	completion chan struct{}
	registered bool
}

func newClusterJSConsumer(address string, eb *clusterJSEventBus) *clusterJSConsumer {
	c := &clusterJSConsumer{
		address:    address,
		eb:         eb,
		completion: make(chan struct{}),
	}
	eb.mu.Lock()
	eb.consumers = append(eb.consumers, c)
	eb.mu.Unlock()
	return c
}

func (c *clusterJSConsumer) Handler(handler MessageHandler) Consumer {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handler = handler
	if c.registered {
		return c
	}
	c.registered = true

	// Publish (fanout): each SERVICE gets a copy; replicas of the same service load-balance (queue group).
	pubSubject := c.eb.subjectPub(c.address)
	pubDurable := "pub_" + sanitizeConsumerName(c.eb.service) + "_" + sanitizeConsumerName(c.address)
	pubGroup := sanitizeConsumerName(c.eb.service)
	pubSub, err := c.eb.js.QueueSubscribe(
		pubSubject,
		pubGroup,
		c.onJSMsg(),
		nats.BindStream(c.eb.streamPub()),
		nats.Durable(pubDurable),
		nats.ManualAck(),
		nats.AckWait(c.eb.ackWait),
		nats.MaxAckPending(c.eb.maxAckPending),
	)
	if err == nil {
		c.subs = append(c.subs, pubSub)
	} else {
		c.eb.logger.Errorf("jetstream subscribe (publish) failed for %s: %v", c.address, err)
	}

	// Send (work-queue): delivered once to one replica (queue group = address).
	sendSubject := c.eb.subjectSend(c.address)
	sendDurable := "send_" + sanitizeConsumerName(c.address)
	sendGroup := sanitizeConsumerName(c.address)
	sendSub, err := c.eb.js.QueueSubscribe(
		sendSubject,
		sendGroup,
		c.onJSMsg(),
		nats.BindStream(c.eb.streamSend()),
		nats.Durable(sendDurable),
		nats.ManualAck(),
		nats.AckWait(c.eb.ackWait),
		nats.MaxAckPending(c.eb.maxAckPending),
	)
	if err == nil {
		c.subs = append(c.subs, sendSub)
	} else {
		c.eb.logger.Errorf("jetstream subscribe (send) failed for %s: %v", c.address, err)
	}

	// Request: core NATS request/reply (queue group = subject).
	reqSubject := c.eb.subjectReq(c.address)
	reqSub, err := c.eb.nc.QueueSubscribe(reqSubject, reqSubject, c.onCoreMsg())
	if err == nil {
		c.subs = append(c.subs, reqSub)
	}

	return c
}

func (c *clusterJSConsumer) Completion() <-chan struct{} { return c.completion }

func (c *clusterJSConsumer) Unregister() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, s := range c.subs {
		_ = s.Unsubscribe()
	}
	c.subs = nil

	select {
	case <-c.completion:
	default:
		close(c.completion)
	}
	return nil
}

func (c *clusterJSConsumer) onJSMsg() nats.MsgHandler {
	return func(nm *nats.Msg) {
		task := concurrency.NewNamedTask(
			"cluster-jetstream-consumer."+c.address,
			func(ctx context.Context) error {
				err := c.handleMsg(nm)
				if err != nil {
					_ = nm.Nak()
					return err
				}
				_ = nm.Ack()
				return nil
			},
		)
		if err := c.eb.executor.Submit(task); err != nil {
			// Backpressure: keep message unacked to be redelivered.
			c.eb.logger.Warnf("cluster consumer overloaded for %s: %v", c.address, err)
		}
	}
}

func (c *clusterJSConsumer) onCoreMsg() nats.MsgHandler {
	return func(nm *nats.Msg) {
		task := concurrency.NewNamedTask(
			"cluster-core-consumer."+c.address,
			func(ctx context.Context) error {
				return c.handleMsg(nm)
			},
		)
		if err := c.eb.executor.Submit(task); err != nil {
			c.eb.logger.Warnf("cluster consumer overloaded for %s: %v", c.address, err)
		}
	}
}

func (c *clusterJSConsumer) handleMsg(nm *nats.Msg) error {
	c.mu.Lock()
	h := c.handler
	c.mu.Unlock()
	if h == nil {
		return nil
	}

	base := c.eb.ctx
	if rid := nm.Header.Get("X-Request-ID"); rid != "" {
		base = WithRequestID(base, rid)
	}
	fctx := newFluxorContext(base, c.eb.vertx)

	headers := make(map[string]string)
	for k, v := range nm.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	msg := &clusterNATSMessage{
		body:         nm.Data,
		headers:      headers,
		replySubject: nm.Reply,
		eb: &clusterNATSEventBus{
			ctx:            c.eb.ctx,
			vertx:          c.eb.vertx,
			nc:             c.eb.nc,
			prefix:         c.eb.prefix,
			requestTimeout: c.eb.requestTimeout,
			executor:       c.eb.executor,
			logger:         c.eb.logger,
		},
	}

	return h(fctx, msg)
}

func sanitizeStreamName(prefix string) string {
	// JetStream stream names are not subjects; keep them simple and stable.
	// Replace '.', '-', and spaces with '_' and uppercase.
	s := strings.TrimSpace(prefix)
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return strings.ToUpper(s)
}

func sanitizeConsumerName(s string) string {
	// Durable/queue names must be stable and conservative.
	// Replace '.' and other separators with '_'.
	x := strings.TrimSpace(s)
	x = strings.ReplaceAll(x, ".", "_")
	x = strings.ReplaceAll(x, "-", "_")
	x = strings.ReplaceAll(x, " ", "_")
	x = strings.ReplaceAll(x, ":", "_")
	x = strings.ReplaceAll(x, "/", "_")
	return x
}
