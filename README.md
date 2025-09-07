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

### QuickStart

```shell
go run github.com/train360-corp/supago
```

### Direct Integration

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
	"fmt"
	"github.com/train360-corp/supago/pkg/services"
	"github.com/train360-corp/supago/pkg/supabase"
	"github.com/train360-corp/supago/pkg/utils"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %v", err))
	}

	// setup custom logger
	if logger, err := utils.NewLogger(zapcore.InfoLevel, false); err != nil {
		panic(err)
	} else {
		utils.OverrideLogger(logger)
	}
	defer utils.Logger().Sync()

	// Create a root ctx that is canceled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// create config to run supabase
	config, err := supabase.GetRandomConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to generate supabase config: %v", err))
	}
	config.DatabaseDataDirectory = filepath.Join(cwd, "data", "postgres")
	config.StorageDirectory = filepath.Join(cwd, "data", "storage")

	// get supabase services from config
	svcs, err := supabase.GetServices(config)
	if err != nil {
		panic(err)
	}

	// run supabase services
	runner, err := services.NewRunner("supago-main-test-net")
	if err != nil {
		panic(err)
	}
	for _, service := range *svcs {
		if err := runner.Run(context.Background(), &service); err != nil {
			runner.Shutdown() // cancel running services
			utils.Logger().Fatal(err)
		}
	}
	utils.Logger().Infof("all services started")
	
	// TODO: 
	// - INTEGRATE/START YOUR OWN SERVER/SERVICES HERE
	// - INTERACT WITH SUPABASE
	// - ETC...

	<-ctx.Done() // block until we get a stop signal

	utils.Logger().Warn("shutdown signal received")

	// shutdown each utility
	runner.Shutdown()

	utils.Logger().Debugf("main context done")
}
```
