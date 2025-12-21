package main

import (
    "context"
    "log"

    "github.com/namtran/creditengine/internal/engine"
)

func main() {
    ctx := context.Background()
    cfg := engine.DefaultConfig()

    svc, err := engine.NewService(cfg)
    if err != nil {
        log.Fatalf("failed to create service: %v", err)
    }

    // run service in foreground (it starts HTTP server and poll loop)
    if err := svc.Run(ctx); err != nil {
        log.Fatalf("service run error: %v", err)
    }
}
