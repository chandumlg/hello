# NATS

NATS is a lightweight, high-performance messaging system for connecting distributed services that solves the problem of getting data reliably and quickly between services without the operational weight of a heavyweight broker.

## Why teams adopt it

Staff, platform, and AI engineers reach for NATS when they need service-to-service or edge-to-cloud communication that is simple to run and fast enough to sit in the hot path — a single static binary with no external dependencies (no ZooKeeper, no JVM), sub-millisecond publish latency, and a client library in nearly every language. The core pub/sub layer is fire-and-forget by design; when durability, replay, or exactly-once-ish delivery is needed, JetStream (NATS's built-in persistence layer) adds streaming, consumer groups, and at-least-once acknowledgment on top of the same server, without introducing a second system to operate.

Common adoption triggers:
- Microservice-to-microservice request/reply and event fan-out where Kafka's operational overhead (brokers, partitions, ZooKeeper/KRaft) is disproportionate to the traffic and latency needs.
- Edge and IoT deployments, or multi-region/multi-cloud topologies, where NATS's built-in clustering and "leaf node" architecture let edge instances bridge back to a central cluster over an unreliable link.
- Real-time systems — chat, gaming, live dashboards, control planes — that need low-latency pub/sub plus simple request/reply semantics (`nats.request()`), not just one-way event streams.
- Replacing ad hoc HTTP polling or a heavyweight queue for internal service coordination when the team wants one small binary to run in dev, CI, and prod identically.
- AI agent and tool-calling systems that need a fast internal message bus between agents/workers without the setup cost of a full event-streaming platform.

## Basic usage

**1. Run a server (single binary, no dependencies):**
```bash
# via Docker
docker run -p 4222:4222 -p 8222:8222 nats:latest -js   # -js enables JetStream

# or natively
nats-server -js
```

**2. Basic pub/sub with the CLI (great for exploring interactively):**
```bash
# Terminal 1: subscribe to a subject
nats sub "orders.*"

# Terminal 2: publish a message
nats pub orders.created '{"order_id": "1234"}'
```

**3. Durable streaming with JetStream (Go client):**
```go
nc, _ := nats.Connect(nats.DefaultURL)
js, _ := jetstream.New(nc)

// Create a stream that persists messages on subject "orders.*"
stream, _ := js.CreateStream(ctx, jetstream.StreamConfig{
    Name:     "ORDERS",
    Subjects: []string{"orders.*"},
})

// Publish — this is now durably stored, not just broadcast
js.Publish(ctx, "orders.created", []byte(`{"order_id": "1234"}`))

// Create a durable, pull-based consumer and process messages with ack
cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
    Durable:   "order-processor",
    AckPolicy: jetstream.AckExplicitPolicy,
})
msgs, _ := cons.Fetch(10)
for msg := range msgs.Messages() {
    process(msg.Data())
    msg.Ack()
}
```

## Common pitfalls

- **Plain pub/sub has no delivery guarantee.** Core NATS is fire-and-forget: if no subscriber is connected when a message is published, it's gone. Reaching for core NATS assuming Kafka-like durability is the single most common mistake — use JetStream whenever a message must survive a subscriber being offline.
- **Subject design is your schema.** NATS routes purely on hierarchical subject strings (`orders.us.created`), and wildcards (`*`, `>`) make fan-out trivial — but a sloppy subject hierarchy designed after the fact is painful to refactor across every publisher and subscriber. Design the subject namespace deliberately up front.
- **JetStream storage and retention need explicit tuning.** Streams default to sensible-but-generic limits; without setting `MaxAge`, `MaxBytes`, or `MaxMsgs`, a high-volume stream can silently grow disk usage until the server hits capacity. Set explicit retention policy per stream.
- **Consumer ack timeouts cause duplicate delivery, not loss.** JetStream is at-least-once: if a consumer doesn't ack within `AckWait`, the message is redelivered. Processing must be idempotent, or duplicate side effects (double-charging, duplicate writes) will occur under redelivery.
- **Clustering requires odd-numbered nodes for quorum** (3 or 5), and JetStream's Raft-based replication means a poorly sized cluster can lose availability on a single node failure — don't run JetStream clustering with 2 nodes expecting HA.
- **It's easy to under-provision monitoring.** NATS is so operationally quiet that teams often skip wiring up the `/varz`, `/connz`, and JetStream monitoring endpoints (or the `nats` CLI's `nats server report`) until a problem is already in progress — set up observability before you need it, not after.
