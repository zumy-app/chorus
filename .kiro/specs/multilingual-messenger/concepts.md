# Key Concepts for Multilingual Messenger

## Database Sharding Concepts

### What is Database Sharding?
**Simple Analogy:** Like splitting a huge library across multiple smaller shelves instead of one giant shelf.

**Technical Definition:** Splitting one large database into multiple smaller databases (shards) distributed across different servers for better performance and scalability.

**Example:**
```
Single Database (Slow):
All 1M users → One PostgreSQL server → Bottleneck!

Sharded Database (Fast):
Users 1-250K → Shard 0
Users 250K-500K → Shard 1  
Users 500K-750K → Shard 2
Users 750K-1M → Shard 3
```

### Shard Key
**Definition:** The field used to decide which shard data goes to.

**Examples:**
- **User ID:** `hash(userID) % 4` → User data goes to specific shard
- **Chat ID:** `hash(chatID) % 4` → All messages for a chat stay together

**Why it matters:** Good shard key = even distribution, bad shard key = hot spots

### Denormalization
**Normal Database (No Duplicates):**
```sql
users: [alice, bob]
chats: [study-group, work-team]
participants: [(alice, study-group), (alice, work-team)]
```

**Denormalized (Intentional Duplicates for Performance):**
```sql
-- Alice's shard contains HER copy of chat info
alice_shard: {
  user: alice,
  chats: [
    {id: "study-group", shard: 0},
    {id: "work-team", shard: 2}
  ]
}
```

**Why denormalize?** Faster queries - look at one shard instead of searching all shards.

## Consistent Hashing Concepts

### What is Consistent Hashing?
**Problem with Regular Hashing:**
```go
// 4 servers
server = hash("data") % 4  // data goes to server 2

// Add 1 server (now 5 total)
server = hash("data") % 5  // data now goes to server 3!
// Problem: ALL data needs to move!
```

**Consistent Hashing Solution:**
Arrange servers in a circle. Data goes to next server clockwise.

```
    Server A (12 o'clock)
         |
Server D  |  Server B
(9)      |      (3)  
         |
    Server C (6 o'clock)
```

**Benefits:** Adding/removing servers only affects nearby data, not everything.

### Virtual Nodes (Detailed)

**The Problem with Basic Consistent Hashing:**
```
Server A: 8% of data
Server B: 67% of data ← Overloaded!
Server C: 25% of data
```

**Virtual Nodes Solution:**
Give each server multiple positions on the circle:

```go
// Each server gets 100 virtual positions
Server A: positions A1, A2, A3, ..., A100
Server B: positions B1, B2, B3, ..., B100  
Server C: positions C1, C2, C3, ..., C100

// Result: Much more even distribution
Server A: ~33.33% of data
Server B: ~33.33% of data
Server C: ~33.33% of data
```

**Real-World Impact:**
- **Without virtual nodes:** 12%-69% distribution (very uneven)
- **With virtual nodes:** 33.31%-33.35% distribution (nearly perfect)

## Distributed Systems Concepts

### Hot Shard
**Definition:** One shard getting way more traffic than others.

**Example:**
```
Shard 0: 100 requests/second (normal)
Shard 1: 150 requests/second (normal)
Shard 2: 5000 requests/second (HOT! 🔥)
Shard 3: 80 requests/second (normal)
```

**Causes:**
- Celebrity creates viral chat → all messages go to same shard
- Popular user → all their chats get heavy traffic
- Trending topic → everyone messaging about it

**Solutions:**
- **Shard splitting:** Split hot shard in half
- **Consistent hashing:** Better data distribution
- **Load balancing:** Spread traffic across servers

### Saga Pattern
**Problem:** Multi-step operations across different databases can fail partially.

**Example (Bank Transfer):**
```go
// Step 1: Subtract $100 from Alice (Bank A) ✅
// Step 2: Add $100 to Bob (Bank B) ❌ FAILS!
// Problem: Alice lost money but Bob didn't get it!
```

**Saga Solution:** For each step, define how to undo it:
```go
steps := []Step{
  {do: subtractFromAlice, undo: addBackToAlice},
  {do: addToBob, undo: subtractFromBob}
}

// If step 2 fails, automatically run undo for step 1
```

### Global Secondary Index (GSI)
**Problem:** Need to search by non-shard-key fields.

**Example:**
- Data sharded by Chat ID
- Want to search by message content: "find messages with 'hello'"
- Problem: Content is not shard key, must search ALL shards

**GSI Solution:** Separate search database:
```sql
-- Main shards (by chat_id)
shard_0: chats A-F
shard_1: chats G-M
shard_2: chats N-S
shard_3: chats T-Z

-- Search index (by content)
search_db: [
  {content: "hello world", message_id: "msg_123", shard: 0},
  {content: "hello there", message_id: "msg_456", shard: 2}
]
```

**Process:**
1. Search index: "hello" → returns msg_123 (shard 0), msg_456 (shard 2)
2. Fetch from only those 2 shards (not all 4!)

### Eventual Consistency
**Strong Consistency:** All updates happen simultaneously everywhere (slower).

**Eventual Consistency:** Updates propagate over time (faster, temporary inconsistencies).

**Example:**
```
Time 0: Alice sends "Hello" to group chat
Time 1: Message stored in chat shard ✅
Time 2: Alice's shard updated ✅
Time 3: Bob's shard updated ✅  
Time 4: Charlie's shard updated ✅
```

Between Time 1-4, system is "eventually consistent" - will be consistent eventually, but not immediately everywhere.

## Performance Concepts

### Cross-Shard Queries
**Problem:** User's data spread across multiple shards.

**Example:**
```
Alice participates in:
- "study-group" chat → Shard 0
- "work-team" chat → Shard 2  
- "family" chat → Shard 1

To get Alice's chats: Query all 3 shards (slow!)
```

**Solution:** Denormalized lookup table on Alice's shard:
```sql
-- Alice's shard contains her chat references
alice_chats: [
  {chat_id: "study-group", shard: 0},
  {chat_id: "work-team", shard: 2},
  {chat_id: "family", shard: 1}
]

// Process:
// 1. Query Alice's shard for chat list (1 query)
// 2. Query only relevant shards (3 queries instead of 4)
```

### Shard Splitting
**When:** Shard gets too much traffic (>1000 QPS, >70% CPU).

**How:** Split data in half using hash function:
```
Before: Shard 2 has chats A,B,C,D → 5000 QPS

After:  
Shard 2: chats A,C (even hash) → 2500 QPS
Shard 4: chats B,D (odd hash) → 2500 QPS
```

**Process:**
1. Create new shard
2. Move ~50% of data to new shard
3. Update routing to use both shards

## Translation Architecture Concepts

### Translation Processing (Phase 1)
**All processing happens on your VPS:**

```
User sends message 
    ↓
Go service on VPS
    ↓
Google Translate API call
    ↓
Cache result in memory
    ↓
Broadcast to recipients via Appwrite Realtime
```

**Benefits:**
- **Simple:** Single service handles everything
- **Cost-effective:** No external message queues
- **Fast:** In-memory caching avoids repeated API calls

### Caching Strategy
**Phase 1 (Simple):**
```go
// In-memory cache in Go service
cache := make(map[string]string)
cacheKey := fmt.Sprintf("%s:%s", text, targetLanguage)
cache[cacheKey] = translatedText
```

**Phase 2 (Advanced):**
```go
// Redis cache shared across multiple Go instances
redis.Set(cacheKey, translatedText, 24*time.Hour)
```

## Scaling Triggers

### When to Scale
**Metrics to Monitor:**
- **CPU usage > 70%** sustained for 10+ minutes
- **Memory usage > 80%** sustained  
- **Response time > 500ms** for 95th percentile
- **Database connections > 80%** of max connections
- **Concurrent users > 5K** active connections

### Scaling Timeline
```
0-1K users: Single VPS + Appwrite (Phase 1)
    ↓
1K-10K users: Add Go service replicas + Redis
    ↓  
10K-100K users: Database sharding + managed PostgreSQL
    ↓
100K+ users: Multi-region + advanced features
```

## State Management Concepts

### Phase 1 (Simple)
```tsx
// React built-in state - no external libraries
function App() {
  const [user, setUser] = useState(null);
  const [chats, setChats] = useState([]);
  const [messages, setMessages] = useState({});
  
  // Simple and effective for MVP
}
```

### Phase 2 (Advanced)
```tsx
// Zustand + TanStack Query for complex state
const useUserStore = create((set) => ({
  user: null,
  setUser: (user) => set({ user }),
}));

const { data: chats } = useQuery(['chats'], fetchChats);
```

**Why wait for Phase 2?** 
- Phase 1: Focus on core functionality
- Phase 2: Add complexity when needed for scaling

These concepts provide the foundation for understanding how the multilingual messenger scales from a simple MVP to an enterprise-grade messaging platform.