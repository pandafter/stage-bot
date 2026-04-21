# Scuderia St4ge - Instagram DM Bot

Automated sales assistant for [Scuderia St4ge](https://scuderiastage.com), a karting academy in Tocancipa, Colombia. The bot handles Instagram DMs with AI-powered intent detection, lead scoring, and strategy-based responses -- converting conversations into course enrollments.

## Features

- **AI Sales Pipeline** -- Intent detection, lead scoring (0-100), and dynamic strategy selection (welcome, inform, persuade, guide, close)
- **Claude AI Integration** -- Natural Colombian Spanish responses with cloned sales team personality
- **Voice Messages** -- ElevenLabs STT/TTS with cloned voice for audio DM support
- **Knowledge Base** -- Google Sheets-powered product info, pricing, FAQ, and objection handling
- **Lead Tracking** -- SQLite-based lead persistence with funnel state management

## Architecture

```
                    Instagram
                       |
                   [Webhook]
                   /       \
              [Text]     [Audio]
                |           |
                |      [STT - ElevenLabs]
                |           |
                +-----+-----+
                      |
                  [AI Brain]
                      |
         +------------+------------+
         |            |            |
   [Intent Detection] | [Lead Scoring]
         |            |            |
         +-----+------+-----+-----+
               |             |
        [Strategy Select] [Knowledge Base]
               |             (Google Sheets)
               |
         [Claude AI] ----> Response
               |
          +----+----+
          |         |
       [Text]   [TTS - ElevenLabs]
          |         |
       Instagram   [Audio Store] --> Instagram
```

## Project Structure

```
instagram-bot/
├── cmd/
│   └── bot/
│       └── main.go                 # Entry point, dependency injection
├── internal/
│   ├── domain/                     # Core types and interfaces
│   │   ├── message.go              # Intent, Strategy, LeadScore, LeadState
│   │   └── interfaces.go           # Messenger, AIEngine, VoiceService, etc.
│   ├── ai/                         # AI/Brain layer
│   │   ├── brain.go                # Orchestrator (Process pipeline)
│   │   ├── claude.go               # Claude API client
│   │   ├── intent.go               # Intent detection (keyword matching)
│   │   ├── intent_test.go          # 56 intent classification tests
│   │   ├── scoring.go              # Lead scoring + strategy selection
│   │   ├── scoring_test.go         # Scoring and strategy tests
│   │   └── prompt.go               # System prompt + strategy instructions
│   ├── messenger/                  # Instagram messaging
│   │   ├── instagram.go            # Graph API client (implements Messenger)
│   │   └── types.go                # Instagram API types
│   ├── voice/                      # Voice processing
│   │   ├── elevenlabs.go           # ElevenLabs TTS/STT (implements VoiceService)
│   │   ├── store.go                # In-memory audio store with TTL
│   │   └── converter.go            # ffmpeg MP3-to-M4A conversion
│   ├── knowledge/                  # Knowledge base
│   │   └── sheets.go               # Google Sheets reader (implements KnowledgeBase)
│   ├── storage/                    # Persistence
│   │   └── sqlite.go               # SQLite with WAL mode
│   ├── webhook/                    # HTTP webhook handling
│   │   ├── handler.go              # Webhook handler (depends on interfaces)
│   │   ├── parser.go               # Payload parser
│   │   ├── parser_test.go          # 8 parser tests
│   │   └── types.go                # Instagram webhook payload types
│   ├── server/                     # HTTP server
│   │   ├── server.go               # Fiber server + routes
│   │   └── middleware.go           # Request logging
│   └── config/                     # Configuration
│       └── config.go               # Environment variable loader
├── docs/                           # Meta required pages
│   ├── index.html
│   ├── privacy-policy.html
│   └── terms-of-service.html
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env.example
└── .gitignore
```

## Setup

### Prerequisites

- Go 1.25+
- ffmpeg (for audio conversion)
- ngrok (for development webhook tunnel)
- SQLite

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `PORT` | No | Server port (default: 8080) |
| `ENV` | No | `development` or `production` (default: development) |
| `APP_ID` | No | Meta App ID |
| `APP_SECRET` | No | Meta App Secret (for HMAC signature validation) |
| `PAGE_ACCESS_TOKEN` | No | Instagram Page Access Token (starts with IGAA...) |
| `INSTAGRAM_ACCOUNT_ID` | No | Instagram Business Account ID |
| `WEBHOOK_VERIFY_TOKEN` | Yes | Custom token for webhook verification |
| `ANTHROPIC_API_KEY` | No | Anthropic API key for Claude AI |
| `ELEVENLABS_API_KEY` | No | ElevenLabs API key for TTS/STT |
| `ELEVENLABS_VOICE_ID` | No | ElevenLabs cloned voice ID |
| `OPENAI_API_KEY` | No | OpenAI API key (Whisper, unused currently) |
| `GOOGLE_SHEET_ID` | No | Google Sheets ID for knowledge base |
| `PUBLIC_URL` | No | Public URL for audio file serving (ngrok URL in dev) |
| `DATABASE_URL` | No | SQLite database path (default: data/bot.db) |
| `TEST_SENDER_ID` | No | Only respond to this user in development |
| `COMMENT_TRIGGER_KEYWORD` | No | Keyword in latest post comments that triggers DM sales flow (default: `piloto`) |

### Local Development

```bash
# 1. Copy environment file
cp .env.example .env
# Edit .env with your credentials

# 2. Start ngrok tunnel
ngrok http 3000

# 3. Update PUBLIC_URL in .env with ngrok URL

# 4. Run the bot
make run
# or
go run ./cmd/bot/
```

### Build

```bash
make build
# Binary output: dist/bot
```

### Test

```bash
make test
```

## Docker

```bash
# Build and run
docker compose up -d

# View logs
docker compose logs -f bot
```

## Sales Pipeline

The AI brain processes each message through a 4-step pipeline:

1. **Intent Detection** -- Keyword-based classification into 16 intents (greeting, price inquiry, buy signal, objection, etc.)
2. **Lead Scoring** -- Cumulative scoring (5 base + intent weight per message) with key signal tracking
3. **Strategy Selection** -- Maps intent + score to 9 strategies (welcome, inform, persuade, guide, close, handle objection, etc.)
4. **Response Generation** -- Claude AI with strategy-aware system prompt + business knowledge context

### Score Thresholds

| Score Range | State | Strategy |
|---|---|---|
| 0-10 | New | Inform |
| 11-30 | Engaged | Inform |
| 31-60 | Interested | Persuade |
| 61-80 | Hot | Guide |
| 81+ | Closing | Close |

## License

Proprietary. All rights reserved.
