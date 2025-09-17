# SupaGo: A Go/Supabase Utility

A lightweight Go utility to help integrate **Supabase** into self-hosted Go servers and projects.  
This project is based on the
official [Supabase Docker Compose guide](https://github.com/supabase/supabase/blob/b741dcb4d58cfc2f45ea9cfa914446b61eb4c1e9/docker/docker-compose.yml),
with a curated subset of services tailored for Go environments.

---

## 🚀 Features

- Provides a **self-hosted Supabase stack** you can spin up with Docker Compose.
- Designed to be integrated into **Go servers/projects** without extra dependencies.
- Includes Supabase core services (Auth, REST, Realtime, Storage, Studio, etc.).
- Excludes unsupported services for now:
    - ❌ Supavisor (connection pooler)
    - ❌ Edge Functions
    - ❌ Vector (observability)

---

## 📦 Services & Versions

The following services (with pinned versions) are included:

| Service                  | Image / Version                          |
|--------------------------|------------------------------------------|
| **Studio**               | `supabase/studio:2025.06.30-sha-6f5982d` |
| **Kong**                 | `kong:2.8.1`                             |
| **Auth** (GoTrue)        | `supabase/gotrue:v2.177.0`               |
| **REST** (PostgREST)     | `postgrest/postgrest:v12.2.12`           |
| **Realtime**             | `supabase/realtime:v2.34.47`             |
| **Storage**              | `supabase/storage-api:v1.25.7`           |
| **Imgproxy**             | `darthsim/imgproxy:v3.8.0`               |
| **Meta** (Postgres Meta) | `supabase/postgres-meta:v0.91.0`         |
| **Analytics** (Logflare) | `supabase/logflare:1.14.2`               |
| **Database**             | `supabase/postgres:15.8.1.060`           |

Unsupported (for now):

- **Supavisor** → `supabase/supavisor:2.5.7`
- **Functions** → `supabase/edge-runtime:v1.69.6`
- **Vector** → `timberio/vector:0.28.1-alpine`

---

## 🛠️ Usage

> Requires docker on the local machine

Directly integrate SupaGo into your freestanding project!

Add the repository:

```shell
go get github.com/train360-corp/supago
```

Run a basic Supabase instance:

```go
package main

import (
	"context"
	"github.com/train360-corp/supago"
	"go.uber.org/zap/zapcore"
	"os/signal"
	"syscall"
)

// encryptionKey is used for SupaBase Vault (must be persisted between restarts)
// Should not be inlined like below, and should be stored/retrieved more securely; consider:
// - supago.EncryptionKeyFromFile (generates a secret from a filepath and reads therefrom)
// - supago.EncryptionKeyFromConfig (like EncryptionKeyFromFile, but generates the key relative to the database directory)
var encryptionKey = supago.StaticEncryptionKey("d9bf2393c65c006cc83625f85a27cc50882a391b1e0ab4fd4c2535dbe1f8a283")

// use any zap logger of choice, customized for specific use-case, or even disable logging altogether:
// var logger = zap.NewNop().Sugar() // No-Op / non-operational logger
var logger = supago.NewOpinionatedLogger(zapcore.InfoLevel, false)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Infof("SupaGo starting")

	sg := supago.New(*supago.NewBaseConfig("example-project", encryptionKey)).
		SetLogger(logger).
		AddServices(supago.Services.All)
	// alternatively, add individual services as needed; e.g.:
	//  AddService(supago.Services.Postgres, supago.Services.Kong)

	if err := sg.RunForcefully(ctx); err != nil {
		logger.Errorf("an error occured while running services: %v", err)
	}

	// TODO: run custom backend services, interact with supabase, etc.

	<-ctx.Done() // block until done

	logger.Warnf("stop-signal recieved")

	sg.Stop() // shutdown services
}

```

