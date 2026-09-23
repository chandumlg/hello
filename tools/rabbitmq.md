# RabbitMQ

RabbitMQ is a message broker that solves the problem of decoupling
services so they can communicate asynchronously through queues instead of
direct, synchronous calls — giving you buffering, routing, and delivery
guarantees without services needing to know about each other or be online
at the same time.

## What problem it solves

When Service A calls Service B directly (HTTP, gRPC), the two are coupled
in time and availability: B has to be up, fast, and able to absorb A's
load right now, or the call fails. That's fine for request/response, but
it's the wrong shape for work that should happen "eventually" — sending an
email, processing an uploaded file, fanning a webhook out to five
downstream consumers, or smoothing a bursty producer against a
slower, rate-limited consumer.

RabbitMQ sits between producers and consumers as a broker: producers
publish messages to an **exchange**, the exchange routes them into one or
more **queues** based on rules (bindings), and consumers pull messages off
those queues independently, at their own pace, acknowledging each one once
it's safely processed. If a consumer is down, messages simply queue up
instead of being lost or blocking the producer. It implements AMQP 0-9-1
natively (plus plugins for MQTT, STOMP, and streams), giving you flexible
routing topologies — direct, fanout, topic, and header-based — rather than
the simple ordered-log model Kafka/Pulsar/Redpanda use.

## Primary use cases and when a team would adopt it

- **Task queues / background jobs** — offload slow or bursty work (image
  resizing, PDF generation, sending notifications) from the request path
  into workers that consume at a sustainable rate.
- **Service decoupling in a microservices architecture** — Service A
  publishes an event ("order placed") without knowing or caring which
  services react to it; new consumers can subscribe later without changing
  the producer.
- **Fan-out / pub-sub** — one event needs to reach several independent
  consumers (billing, analytics, notifications) via a fanout or topic
  exchange, each with its own queue and processing rate.
- **Work distribution across a worker pool** — multiple consumer instances
  compete for messages on the same queue, giving you load-balanced,
  horizontally scalable processing with per-message acknowledgment.
- **RPC-style request/reply over messaging** — using reply-to queues and
  correlation IDs when you want messaging's resilience but still need a
  response back to the caller.

Reach for RabbitMQ when you need flexible routing and per-message
work-queue semantics (competing consumers, retries, dead-lettering) more
than you need a durable, replayable, ordered log of everything that ever
happened — that's the point where Kafka/Redpanda/Pulsar (already covered
in this series) tend to be the better fit instead.

## Basic usage

**1. Run RabbitMQ locally (with the management UI):**

```bash
docker run -d --name rabbitmq \
  -p 5672:5672 -p 15672:15672 \
  rabbitmq:4-management
```

The management UI is now at `http://localhost:15672` (default login
`guest`/`guest`, local-only by default) and shows queues, exchanges,
connections, and message rates in real time.

**2. Publish and consume with Python (`pika`):**

```python
# producer.py
import pika

conn = pika.BlockingConnection(pika.ConnectionParameters("localhost"))
ch = conn.channel()
ch.queue_declare(queue="emails", durable=True)

ch.basic_publish(
    exchange="",
    routing_key="emails",
    body="send-welcome-email:user-42",
    properties=pika.BasicProperties(delivery_mode=2),  # persist to disk
)
conn.close()
```

```python
# consumer.py
import pika

conn = pika.BlockingConnection(pika.ConnectionParameters("localhost"))
ch = conn.channel()
ch.queue_declare(queue="emails", durable=True)
ch.basic_qos(prefetch_count=1)  # don't hand a worker more than 1 unacked message

def callback(ch, method, properties, body):
    print(f"processing: {body.decode()}")
    ch.basic_ack(delivery_tag=method.delivery_tag)  # only after success

ch.basic_consume(queue="emails", on_message_callback=callback)
ch.start_consuming()
```

**3. Route events to multiple queues with a topic exchange:**

```python
ch.exchange_declare(exchange="orders", exchange_type="topic")

# Two independent consumers bind their own queues with different patterns
ch.queue_bind(exchange="orders", queue="billing_q", routing_key="order.placed.*")
ch.queue_bind(exchange="orders", queue="analytics_q", routing_key="order.*.*")

ch.basic_publish(exchange="orders", routing_key="order.placed.us",
                  body="order-1001")
# billing_q and analytics_q both receive it; a queue bound to "order.shipped.*"
# would not
```

## Common pitfalls

- **Unbounded queues cause memory/disk pressure.** A consumer that's down
  or too slow lets a queue grow without limit by default. Set queue
  length limits, TTLs, or dead-letter exchanges (`x-max-length`,
  `x-message-ttl`, `x-dead-letter-exchange`) so a stuck consumer degrades
  gracefully instead of taking down the broker.
- **Forgetting `durable=True` / persistent delivery mode.** A queue
  declared without `durable=True`, or a message published without
  `delivery_mode=2`, is lost on broker restart. Both the queue and the
  message need to opt into durability.
- **Auto-ack throws away your delivery guarantee.** Consuming with
  auto-ack means a message is considered "delivered" the instant it's
  sent, even if the consumer crashes mid-processing — you silently lose
  messages. Ack manually, after the work is actually done.
- **No prefetch limit starves other consumers.** Without
  `basic_qos(prefetch_count=...)`, RabbitMQ can hand one fast-connecting
  consumer a large batch of messages while others sit idle, defeating
  load balancing across a worker pool.
- **It's not a log — messages are gone once consumed.** Unlike
  Kafka-style brokers, RabbitMQ queues are destructive: once a message is
  acked and there's no other consumer bound to see it, it's gone. Don't
  reach for it when you need replay, long retention, or multiple
  independent consumer groups replaying history at their own offsets.
- **Clustering and quorum queues need deliberate setup.** Classic mirrored
  queues are deprecated; production HA setups should use **quorum
  queues**, which have different memory and performance characteristics
  than classic queues — test under realistic load before assuming
  parity.
