# Local AI Roadmap

Learning roadmap for integrating a local AI (Ollama) into places. The goal is to learn local AI concepts hands-on using places as a sandbox project.

## Prerequisites

- [Ollama](https://ollama.com/) installed and running
- A small model pulled: `ollama pull llama3.2:3b` (2GB, fast on CPU)
- For embeddings: `ollama pull nomic-embed-text` (270MB)

## Level 1: Ollama Basics

**Goal:** Understand how local LLM inference works.

### Steps

1. Install Ollama, verify with `ollama list`
2. Chat via CLI: `ollama run llama3.2:3b`
3. Learn the REST API:

```bash
# Generate (single prompt, no conversation)
curl http://localhost:11434/api/generate -d '{
  "model": "llama3.2:3b",
  "prompt": "What is a workspace navigator?",
  "stream": false
}'

# Chat (multi-turn conversation)
curl http://localhost:11434/api/chat -d '{
  "model": "llama3.2:3b",
  "messages": [{"role": "user", "content": "What is Go?"}],
  "stream": false
}'
```

4. Experiment with system prompts to control behavior
5. Try streaming vs non-streaming responses

### What you learn
- LLM inference latency (cold start vs warm)
- Token generation speed on your hardware
- Context window limits
- System prompt engineering basics

## Level 2: RAG with Places

**Goal:** Make the AI understand your places by injecting them as context.

### Concept

RAG = Retrieval-Augmented Generation. Instead of training the model, you include relevant data in the prompt. The model reasons over it at inference time.

### Implementation Plan

1. **Go client for Ollama** — HTTP calls to `localhost:11434/api/chat`
2. **Build context** — serialize all places (name, path, tags, note) into a compact text block
3. **System prompt** — instruct the model to match user queries to places:

```
You are a workspace navigator assistant. You know these places:
- boss-api: C:\dev\repos\..., tags: game, work, note: Boss of the World backend API
- horses: C:\dev\repos\..., tags: game, work, note: Horse ranch manager
- places: C:\dev\repos\..., tags: cli, private, tool
...

When the user describes a project, respond with ONLY the place name that best matches.
If no match, respond with "none".
```

4. **API endpoint** — `POST /api/ai-search` accepts a query, returns matched place name
5. **Dashboard integration** — search bar that uses AI matching alongside existing text filter

### Architecture

```
Dashboard                  Go Server                 Ollama
   |                          |                         |
   |  POST /api/ai-search     |                         |
   |  { "query": "the horse   |                         |
   |    breeding game" }       |                         |
   |------------------------->|  Build prompt with       |
   |                          |  all places as context   |
   |                          |  POST /api/chat          |
   |                          |------------------------->|
   |                          |  { "horses" }            |
   |                          |<-------------------------|
   |  { "match": "horses" }   |                         |
   |<-------------------------|                         |
```

### What you learn
- RAG pattern (context injection vs fine-tuning)
- Prompt engineering for structured output
- Handling latency in UI (loading states, streaming)
- When RAG is enough vs when you need embeddings

## Level 3: Embeddings & Semantic Search

**Goal:** Match queries by meaning, not just keywords.

### Concept

Embeddings convert text into high-dimensional vectors. Similar meanings produce similar vectors. "horse breeding game" and "horses" have similar embeddings even though they share few words.

### Implementation Plan

1. **Generate embeddings** for each place:

```bash
curl http://localhost:11434/api/embed -d '{
  "model": "nomic-embed-text",
  "input": "horses - horse ranch manager, tags: game work, C:\\dev\\repos\\jumpgate\\horse_ranch_manager"
}'
# Returns: { "embeddings": [[0.123, -0.456, ...]] }  (768 dimensions)
```

2. **Store in SQLite** — new table in `sessions.db` or separate `embeddings.db`:

```sql
CREATE TABLE embeddings (
    place TEXT PRIMARY KEY,
    vector BLOB NOT NULL,  -- float32 array serialized
    updated_at INTEGER
);
```

3. **Cosine similarity search** in Go:

```go
func cosineSimilarity(a, b []float32) float32 {
    var dot, normA, normB float32
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dot / (sqrt(normA) * sqrt(normB))
}
```

4. **Re-index on change** — regenerate embedding when place name/tags/note changes
5. **API endpoint** — `POST /api/ai-search` embeds the query and finds nearest place

### What you learn
- Vector embeddings and similarity metrics
- Embedding model selection (nomic-embed-text vs mxbai-embed-large)
- Index management (when to re-embed)
- Trade-offs: embedding search is faster but less flexible than RAG chat

## Level 4: Advanced Features

### Git Diff Summary

Use the LLM to summarize `git status --porcelain` output into human-readable text.

```
Input:  M internal/app/app.go, M internal/app/static/index.html, A docs/roadmap.md
Output: "Modified API server and dashboard frontend, added AI roadmap doc"
```

Show in the git dirty tooltip instead of raw file list.

### Smart Place Notes

When adding a new place, auto-generate a note by reading the project's README.md or directory structure:

```
Input:  README.md content from C:\dev\repos\jumpgate\horse_ranch_manager
Output: "Multiplayer horse breeding game — Go backend with WebSocket API"
```

### Streaming Responses

For chat-like features, stream tokens to the UI for perceived speed:

```go
resp, _ := http.Post("http://localhost:11434/api/chat", "application/json", body)
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    var chunk struct { Message struct { Content string } }
    json.Unmarshal(scanner.Bytes(), &chunk)
    // Send to frontend via SSE or WebSocket
}
```

### Model Comparison

Test different models for your use case:

| Model | Size | Speed | Quality |
|-------|------|-------|---------|
| llama3.2:1b | 1.3GB | fastest | basic matching |
| llama3.2:3b | 2GB | fast | good for RAG |
| mistral:7b | 4GB | medium | better reasoning |
| nomic-embed-text | 270MB | instant | embeddings only |

### Function Calling

Some models support tool/function calling — the model decides which function to invoke:

```json
{
  "tools": [{
    "type": "function",
    "function": {
      "name": "open_place",
      "description": "Open a place in the workspace navigator",
      "parameters": { "name": "string", "action": "string" }
    }
  }]
}
```

User says "open Claude at the horse project" → model calls `open_place("horses", "claude")`.

## Detection & Feature Gating

The places-app should detect Ollama availability and gate AI features:

```go
func hasOllama() bool {
    client := &http.Client{Timeout: 500 * time.Millisecond}
    resp, err := client.Get("http://localhost:11434/api/version")
    if err != nil { return false }
    resp.Body.Close()
    return resp.StatusCode == 200
}
```

Expose `has_ollama` in `/api/places` response. Dashboard shows/hides AI search bar accordingly.

## Resources

- [Ollama docs](https://github.com/ollama/ollama/blob/main/docs/api.md) — REST API reference
- [nomic-embed-text](https://ollama.com/library/nomic-embed-text) — embedding model
- [RAG explained](https://www.pinecone.io/learn/retrieval-augmented-generation/) — concept overview
