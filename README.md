# ReadPanda

A virtual book club platform that allows readers to create private rooms to read and discuss books together with friends.

## Overview

ReadPanda is a full-stack web application designed to create a "virtual book club" experience. Users can form private or public groups (called "rooms") to read books together, with the ability to comment on specific parts of a book. Comments are visible only to members of that room, fostering a shared reading experience without public spoilers.

## Problem Statement

**Who is the user?**
The user is someone who enjoys reading and wants to be part of a book club with their friends, but is unable to figure out a set time due to scheduling, location and reading pace conflicts. They want the enjoyment of reading a book together, and commenting on their favourite parts but only with their closest friends, not publicly.

**What problem are we solving?**
The problem being solved is one of creating an app where one can create "rooms" to read a book with a select group of friends. Currently, reading apps, when you annotate, highlight, or comment on a book, showcase that publicly. In this, whatever you comment will be visible only privately to a select group of people.

This can be used to have virtual, long distance book clubs, reconnect with people over love of books.

There can also be public rooms that one can create, so they can connect with people over a book they like. Instead of having to go through book groups to find people who have interest in the book, the book is the central part around which people connect. Each room can be set to a limit, ranging till 20 people, to keep the rooms as close knit as possible, instead of becoming too huge to control.

**Why does it matter?**
It is a way to help people come together around books again, a virtual yet private platform. Currently every platform that can be used for this purpose is only public, or tertiary platforms like discord, instagram or reddit. This combines both the purposes of a book sharing platform and a group platform.

## Architecture

ReadPanda uses a monorepo architecture with separate frontend and backend packages:

- **Frontend (`packages/portal`):** React-based SPA built with Vite
- **Backend:** Two implementations available:
  - **Node.js (`packages/api`):** Original JavaScript/Express implementation
  - **Go (`packages/api-go`):** High-performance Go implementation with identical API

Both backend implementations share the same PostgreSQL database and Firebase Storage, and the frontend works seamlessly with either one.

### 🚀 Why Two Backends?

**Performance Benefits of Go Backend:**
- ⚡ **3x faster** response times
- 💾 **80% less memory** usage (10MB vs 50MB baseline)
- 💰 **60% lower costs** - handle 4x more users per server
- 🏃 **10x faster** cold starts (crucial for autoscaling)
- 📦 **90% smaller** deployments (24MB vs 200MB+)
- 🛡️ **More reliable** - better uptime and stability

**When to use which:**
- Use **Node.js** for rapid development and prototyping
- Use **Go** for production deployments with better performance and lower costs

> 📊 **[See detailed performance analysis →](PERFORMANCE_BENEFITS.md)**

## Key Features

- 📚 Create private and public reading rooms
- 💬 Comment on specific parts of books
- 🔖 Track reading progress with bookmarks
- 👥 Invite friends to reading rooms
- 🔐 Secure authentication (email/password and Google OAuth)
- 📤 Upload and share books
- 🎨 User preferences and customization

## Technology Stack

### Frontend
- React 18
- Vite
- Tailwind CSS
- React Router

### Backend (Choose one)
- **Node.js:** Express, PostgreSQL, Firebase Admin SDK
- **Go:** Gorilla Mux, PostgreSQL, Cloud Storage SDK

### Database & Storage
- PostgreSQL
- Firebase Cloud Storage

## Getting Started

See [TECHNICAL_DOCUMENTATION.md](TECHNICAL_DOCUMENTATION.md) for detailed setup instructions.

### Quick Start

1. Clone the repository:
```bash
git clone <repository-url>
cd readpanda
```

2. Choose your backend:

**Option A: Node.js Backend**
```bash
cd packages/api
npm install
npm start
```

**Option B: Go Backend**
```bash
cd packages/api-go
go mod download
go run cmd/server/main.go
```

3. Start the frontend:
```bash
cd packages/portal
npm install
npm run dev
```

## Documentation

- **[Performance Benefits Guide](PERFORMANCE_BENEFITS.md)** - 🚀 How Go improves your app performance
- [Technical Documentation](TECHNICAL_DOCUMENTATION.md) - Complete architecture and setup guide
- [Node.js vs Go Comparison](NODEJS_VS_GO_COMPARISON.md) - Detailed comparison of both backend implementations
- [API README](packages/api/README.md) - Node.js backend documentation
- [Go API README](packages/api-go/README.md) - Go backend documentation

## Performance Testing

Run the included benchmark script to compare both backends:

```bash
# Make sure both backends are running
./benchmark.sh
```

For detailed load testing:
```bash
# Using Apache Bench
ab -n 1000 -c 100 http://localhost:3000/api/v1/books/all

# Using wrk (more advanced)
wrk -t4 -c100 -d30s http://localhost:3000/api/v1/books/all
```

## User Stories

### Define the User Persona (Who?)

A frequent reader who loves books and wants to share their thoughts throughout the book with their friends. But using tertiary apps makes it harder, as either the friend hasn't reached that specific part of the book and is met with spoilers, or is unable to recall exactly what they are talking about.

Read Panda will allow there be to conversation and comments at specific parts of the book, being a convergence between social media and book reading platforms.

### Capture the Goal or Action (What?)
I want to be able to read a book with my friends.
There will be bookmarks to mark till where each person has read.
When I comment on a specific part of a book I want only the people in my reading room to see them, not publicly.

### Define the Value/Benefit (Why?)
The main point of reading clubs and books has been to bring together a group of people who read the same book and discuss it, sharing in the joy of it. Reading groups/book groups nowadays are slewed, especially nowadays, where the central idea of reading the same book has changed into recommendations, and maybe a few comments on a similarly read book.

Read Panda aims to bring the focus back onto books, creating small private spaces for select people, where groups are formed around the book. Capping the amount of people in a room allows there to be wholesome communication about the book.

## License

MIT
