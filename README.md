# QueueDrop

A fair-queue ticket-drop system — the backend behind a flash sale that can't
be gamed by refreshing faster or spamming requests. Built to solve the real
problem behind Ticketmaster-style "waiting room" queues: thousands of buyers
hit "join queue" in the same second, but only a handful of tickets exist.

**Live app:** https://queue-drop.vercel.app/
**Live API:** https://queuedrop-production-d25f.up.railway.app/

## Tech Stack
- **Backend:** Go + Fiber — deployed on Railway
- **Database:** MongoDB (users, events, orders)
- **Queue:** RabbitMQ — fair, ordered, durable request processing
- **Cache / Atomic operations:** Redis — atomic stock reservation
- **Real-time:** gofiber/websocket — live per-user order notifications
- **Auth:** JWT (golang-jwt) + bcrypt, with role-based access control
- **Frontend:** Vanilla HTML/CSS/JS — deployed on Vercel

## The Core Problem This Solves

A naive "check stock, then decrement" approach has a race condition: two
requests can both read "1 ticket left" at the same instant and both believe
they succeeded, overselling. QueueDrop avoids this two ways at once:

1. **RabbitMQ decouples "accept the request" from "process the request."**
   `POST /api/events/:id/queue` publishes a message and returns instantly —
   it never touches stock directly. A single worker consumes the queue
   **one message at a time, in strict arrival order**, so there's no
   concurrent access to worry about at the processing stage.
2. **Redis's atomic operations remove any remaining race window.** Stock
   reservation is a single atomic operation against Redis — read-and-decrement
   happens indivisibly, so even under extreme concurrent load, exactly as
   many requests succeed as there are tickets, no more.

## Roles

| Role | Can do |
|---|---|
| **User** | Browse drops, join a queue, purchase |
| **Organizer** | Everything a User can, plus create/edit their own drops |
| **Admin** | Everything, on every drop, regardless of owner |

Role checks are enforced in middleware; ownership checks (an organizer editing
*their own* drop, not someone else's) are enforced inside the handler after
looking up the resource — those are two different kinds of checks and can't
both live in generic middleware.

## Architecture

```
POST /api/events/:id/queue (Fiber, JWT-authed)
     │
     ▼
Publish to RabbitMQ "purchase_queue"  ──►  API responds instantly
     │
     ▼
Worker consumes ONE message at a time, in order
     │
     ▼
Redis: atomic stock reservation — reserve a ticket or fail (no race condition)
     │
     ├─ success ──► MongoDB: Order{status: "confirmed"}
     └─ fail     ──► MongoDB: Order{status: "sold_out"}
     │
     ▼
Result available two ways:
  • GET /api/events/:id/status — polled by the frontend right after joining
  • WS  /ws/notifications      — pushed instantly if the client is connected
```

The frontend currently gets its live result via **polling** the status
endpoint, with the WebSocket push endpoint available and working on the
backend as a lower-latency alternative — a deliberate two-path design, since
polling degrades gracefully (works even if a WebSocket connection drops)
while the WebSocket gives near-instant delivery when connected.

## API Routes

| Method | Route | Auth | Description |
|---|---|---|---|
| POST | `/api/register` | No | Create account (`user` or `organizer` role) |
| POST | `/api/login` | No | Log in, returns JWT |
| GET | `/api/events` | No | List all drops |
| POST | `/api/events` | Organizer/Admin | Create a drop, sets Redis stock count |
| PUT | `/api/events/:id` | Organizer (own)/Admin | Edit a drop |
| POST | `/api/events/:id/queue` | Any logged-in user | Join the queue — publishes to RabbitMQ, returns instantly |
| GET | `/api/events/:id/status` | Any logged-in user | Poll this user's order status for one event (`pending`/`confirmed`/`sold_out`) |
| GET | `/api/orders/me` | Any logged-in user | This user's full order history |
| GET | `/ws/notifications?token=<jwt>` | Any logged-in user | Live push of order results |

## Project Structure
```
QueueDrop/
├── Backend/
│   ├── main.go
│   ├── handlers/
│   │   ├── auth_handlers.go
│   │   ├── event_handlers.go
│   │   ├── order_handlers.go
│   │   └── notification_handlers.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── role.go
│   │   └── ws_auth.go
│   ├── models/
│   │   ├── users.go
│   │   ├── events.go
│   │   └── orders.go
│   ├── database/
│   │   └── mongo.go
│   ├── cache/
│   │   └── redis.go
│   ├── queue/
│   │   ├── rabbitmq.go
│   │   └── worker.go
│   └── notify/
│       └── hub.go
├── Frontend/
│   ├── index.html
│   ├── app.js
│   └── style.css
└── README.md
```

## How to Run Locally

**1. Start MongoDB, Redis, and RabbitMQ (Docker):**
```bash
docker run --name mongo-queuedrop -p 27018:27017 -d mongo
docker run --name redis-queuedrop -p 6380:6379 -d redis
docker run --name rabbitmq-queuedrop -p 5672:5672 -p 15672:15672 -d rabbitmq:management
```

**2. Configure environment variables** — create `Backend/.env`:
```
JWT_SECRET=your-secret-here
MONGODB_URI=mongodb://127.0.0.1:27018
REDIS_ADDR=127.0.0.1:6380
RABBITMQ_URL=amqp://guest:guest@127.0.0.1:5672/
PORT=8080
```

**3. Run the backend:**
```bash
cd Backend
go run main.go
```

**4. Open the frontend:**
Open `Frontend/index.html` directly, or point `localStorage["queuedrop_api_base"]`
at your local backend instead of the deployed Railway URL.

## Deployment Notes

- **Backend (Railway):** connects to managed cloud services rather than
  Docker containers — `REDIS_URL` (a full `rediss://` connection string, for
  providers like Redis Cloud/Upstash that require TLS + auth) takes priority
  over the bare `REDIS_ADDR` used for local dev; `MONGODB_URI` points at
  MongoDB Atlas; `RABBITMQ_URL` points at a hosted broker (e.g. CloudAMQP).
  `PORT` is read from the environment, since Railway assigns it dynamically.
- **Frontend (Vercel):** static deploy, `DEFAULT_API_BASE` points at the
  live Railway backend URL, overridable via `localStorage` for local testing
  against a local backend.

## What I Learned
- **Message queues vs. Redis Pub/Sub:** RabbitMQ persists messages until
  consumed and guarantees ordering — a genuinely different tool from Redis
  Pub/Sub, which drops anything published while nobody's listening. Queues
  earn their place specifically when *fair, ordered processing* matters more
  than *instant fan-out*.
- **Atomic operations as a correctness guarantee, not just a performance
  trick:** atomic stock reservation in Redis isn't about speed — it's the
  actual mechanism that prevents overselling under real concurrent load,
  something a plain "read then write" can never guarantee.
- **Role checks vs. ownership checks are different problems:** role
  middleware answers "can this kind of user even attempt this," while
  ownership checks require loading the specific resource first and belong
  inside the handler — trying to force both into generic middleware doesn't
  work cleanly.
- **Designing for two different infrastructures at once (local Docker vs.
  managed cloud services):** connection code that checks for a full URL
  (`REDIS_URL`, provided by managed hosts) before falling back to a bare
  host:port (for local dev) is a pattern that shows up constantly in real
  deployments — the same codebase has to work against very different
  environments without code changes.
- **A long, genuinely difficult debugging arc** across this project —
  a case-insensitive folder collision (`Handlers` vs `handlers`), a queue
  name typo that silently dropped every published message with no error
  anywhere, a missing `.Hex()` conversion that made Redis cache keys never
  match between writer and reader, and a completely missing
  `godotenv.Load()` call that meant an entire `.env` file was silently
  never read. Every one of these produced no compiler error and no obvious
  symptom beyond "nothing happens" — tracing each back to its root cause by
  comparing what was actually written to disk against what was assumed to
  be there was the real skill exercised throughout this project.

## Future Improvements
- Real payment integration (Stripe) — currently mocked, since it's a
  separate concern from this project's core focus on fair queueing under
  contention
- Wire the frontend to consume `/ws/notifications` directly instead of
  relying solely on polling, for lower-latency updates
- Multiple concurrent workers with partitioned queues, for higher throughput
- Admin dashboard: view all orders, manually resolve disputes
- Waiting-room-style live queue position updates (not just the final result)