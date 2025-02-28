package loki

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

//thanks to https://github.com/paul-milne/zap-loki

type Logger interface {
	Error(msg string, args ...any)
}

type Config struct {

	// TenantValue is the value associated with the tenant for multi-tenant systems.
	// It is optional. If not provided, the request will not include a tenant header.
	TenantValue string

	// TenantKey is the key used to specify the tenant in the request headers.
	// It is optional. If not provided, the request will not include a tenant header.
	TenantKey string

	// Url of the loki server, e.g. https://example-prod.grafana.net/loki/api/v1/push
	Url string `validate:"required"`

	// BatchMaxSize is the maximum number of log lines that are sent in one request
	BatchMaxSize int `validate:"gte=1"`

	// BatchMaxWait is the maximum time to wait before sending a request
	BatchMaxWait time.Duration `validate:"gte=1"`

	// Labels that are added to all log lines
	Labels map[string]string

	// Username is the username used for basic authentication when pushing logs to Loki.
	// It is optional. If authentication is not required, leave it empty.
	Username string

	// Password is the password associated with the Username for basic authentication.
	// It is optional. If authentication is not required, leave it empty.
	Password string
}

func (cfg *Config) setDefaults() {
	if cfg.BatchMaxSize == 0 {
		cfg.BatchMaxSize = 1000
	}
	if cfg.BatchMaxWait == 0 {
		cfg.BatchMaxWait = 5 * time.Second
	}
	if cfg.Labels == nil {
		cfg.Labels = map[string]string{}
	}
}

type Pusher struct {
	config    *Config
	ctx       context.Context
	cancel    context.CancelFunc
	client    *http.Client
	quit      chan struct{}
	entry     chan LogEntry
	waitGroup sync.WaitGroup
	logsBatch []streamValue
	logger    Logger
}

type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
	Caller  string `json:"caller"`
}

type lokiPushRequest struct {
	Streams []stream `json:"streams"`
}

type stream struct {
	Stream map[string]string `json:"stream"`
	Values []streamValue     `json:"values"`
}

type streamValue []string

func New(ctx context.Context, cfg Config, logger Logger) (*Pusher, error) {

	cfg.setDefaults()
	err := validator.New().Struct(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	p := &Pusher{
		config:    &cfg,
		ctx:       ctx,
		cancel:    cancel,
		client:    &http.Client{},
		quit:      make(chan struct{}),
		entry:     make(chan LogEntry),
		logsBatch: make([]streamValue, 0, cfg.BatchMaxSize),
		logger:    logger,
	}

	p.waitGroup.Add(1)
	go p.run()
	return p, nil
}

// Push pushes log to loki
func (p *Pusher) Push(e LogEntry) error {
	p.entry <- e
	return nil
}

// Stop stops the loki pusher
func (p *Pusher) Stop() {
	close(p.quit)
	p.waitGroup.Wait()
	p.cancel()
}

func (p *Pusher) run() {
	ticker := time.NewTicker(p.config.BatchMaxWait)
	defer ticker.Stop()

	trySendBatch := func() {
		err := p.send()
		if err != nil {
			p.logger.Error("failed to send logs", "error", err)
		}
		p.logsBatch = p.logsBatch[:0]
	}

	defer func() {
		if len(p.logsBatch) > 0 {
			trySendBatch()
		}

		p.waitGroup.Done()
	}()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.quit:
			return
		case entry := <-p.entry:
			p.logsBatch = append(p.logsBatch, newLog(entry))
			if len(p.logsBatch) >= p.config.BatchMaxSize {
				trySendBatch()
			}
		case <-ticker.C:
			if len(p.logsBatch) > 0 {
				trySendBatch()
			}
		}
	}
}

func newLog(entry LogEntry) streamValue {
	entryJson, err := json.Marshal(entry)
	if err != nil {
		return nil
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	return []string{timestamp, string(entryJson)}
}

func (p *Pusher) send() error {
	buf := bytes.NewBuffer([]byte{})
	gz := gzip.NewWriter(buf)

	if err := json.NewEncoder(gz).Encode(lokiPushRequest{Streams: []stream{{
		Stream: p.config.Labels,
		Values: p.logsBatch,
	}}}); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, p.config.Url, buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if len(p.config.TenantKey) > 0 {
		req.Header.Set(p.config.TenantKey, p.config.TenantValue)
	}
	req.WithContext(p.ctx)

	if p.config.Username != "" && p.config.Password != "" {
		req.SetBasicAuth(p.config.Username, p.config.Password)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("recieved unexpected response code from Loki: %s, body: %s", resp.Status, string(body))
	}

	return nil
}
