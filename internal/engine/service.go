package engine

import (
    "context"
    "database/sql"
    "html/template"
    "log"
    "net/http"
    "time"

    _ "github.com/lib/pq"
    "github.com/namtran/creditengine/internal/chain"
    "github.com/namtran/creditengine/internal/store"
)

// Service is a small orchestrator that polls deposits and credits accounts when final.
type Service struct {
    cfg   *Config
    db    *sql.DB
    store *store.Store
    chain chain.ChainClient
}

// NewService constructs a Service with real DB and optional chain client.
func NewService(cfg *Config) (*Service, error) {
    db, err := sql.Open("postgres", cfg.PostgresDSN)
    if err != nil {
        return nil, err
    }
    st := store.New(db)
    ch, _ := chain.New(cfg.RPCUrl)
    return &Service{cfg: cfg, db: db, store: st, chain: ch}, nil
}

// NewServiceWithStore creates a Service with an injected store and chain client (testable).
func NewServiceWithStore(cfg *Config, s *store.Store, ch chain.ChainClient) *Service {
    return &Service{cfg: cfg, db: nil, store: s, chain: ch}
}

// Run starts a tiny HTTP UI and a background poll loop.
func (s *Service) Run(ctx context.Context) error {
    mux := http.NewServeMux()
    tmpl := template.Must(template.New("ui").Parse(indexHTML))
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        deposits, _ := s.store.ListDeposits(r.Context())
        _ = tmpl.Execute(w, deposits)
    })
    srv := &http.Server{Addr: ":8080", Handler: mux}
    go srv.ListenAndServe()

    ticker := time.NewTicker(s.cfg.PollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := s.ProcessOnce(ctx); err != nil {
                log.Printf("process once error: %v", err)
            }
        }
    }
}

// ProcessOnce processes pending deposits: consults chain (if provided), updates DB and credits idempotently.
func (s *Service) ProcessOnce(ctx context.Context) error {
    deposits, err := s.store.GetPendingDeposits(ctx)
    if err != nil {
        return err
    }
    for _, d := range deposits {
        if s.chain == nil {
            if d.Confirmations >= s.cfg.Confirmations {
                if err := s.store.CreditIfNotCredited(ctx, d); err != nil {
                    log.Printf("failed to credit deposit %s: %v", d.TxHash, err)
                }
            }
            continue
        }

        txBlock, conf, blockHash, found, reverted, err := s.chain.ConfirmationsFromTxHash(ctx, d.TxHash)
        if err != nil {
            log.Printf("chain error: %v", err)
            continue
        }

        if !found {
            if err := s.store.MarkDepositReorged(ctx, d.ID); err != nil {
                log.Printf("failed to mark reorged for %s: %v", d.TxHash, err)
            }
            continue
        }

        // compare nullable block hashes when available
        var dBlockHash string
        if d.BlockHash.Valid {
            dBlockHash = d.BlockHash.String
        }
        if dBlockHash != "" && blockHash != "" && dBlockHash != blockHash {
            if err := s.store.MarkDepositReorged(ctx, d.ID); err != nil {
                log.Printf("failed to mark reorged for %s: %v", d.TxHash, err)
            }
            continue
        }

        // update tx info (txBlock is a uint64 from chain; store.UpdateDepositTxInfo accepts uint64)
        if err := s.store.UpdateDepositTxInfo(ctx, d.ID, txBlock, blockHash); err != nil {
            log.Printf("failed to update tx info for %s: %v", d.TxHash, err)
        }
        if err := s.store.UpdateDepositConfirmations(ctx, d.ID, conf); err != nil {
            log.Printf("failed to update confirmations for %s: %v", d.TxHash, err)
        }
        if reverted {
            if err := s.store.MarkDepositReorged(ctx, d.ID); err != nil {
                log.Printf("failed to mark reorged for %s: %v", d.TxHash, err)
            }
            continue
        }
        if conf >= s.cfg.Confirmations {
            if err := s.store.CreditIfNotCredited(ctx, d); err != nil {
                log.Printf("failed to credit deposit %s: %v", d.TxHash, err)
            }
        }
    }
    return nil
}

// Minimal index page used by the (optional) HTTP UI.
const indexHTML = `<html><body><h1>Deposits</h1>{{range .}}<div>{{.ID}} {{.TxHash}} {{.Status}}</div>{{end}}</body></html>`
