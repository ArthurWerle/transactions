package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ArthurWerle/transactions/internal/config"
	"github.com/ArthurWerle/transactions/internal/model"
)

// RebuildInsightsJobType is the job enqueued on the events service when a
// significant transaction invalidates the cached spending insight.
const RebuildInsightsJobType = "rebuild-spending-insights"

// InsightsNotifier invalidates the AI spending insight cache when a
// transaction is significant enough to change the analysis. Notifications are
// fire-and-forget: they never block or fail the transaction that triggered
// them.
type InsightsNotifier interface {
	TransactionCreated(tx *model.Transaction)
}

// NewInsightsNotifier returns a notifier that enqueues a rebuild job on the
// events service, or a no-op notifier when no events base URL is configured.
func NewInsightsNotifier(cfg config.InsightsConfig, transactionService TransactionsService, logger *slog.Logger) InsightsNotifier {
	if cfg.EventsBaseURL == "" {
		logger.Info("insights notifier disabled: EVENTS_BASE_URL not set")
		return noopInsightsNotifier{}
	}
	return &eventsInsightsNotifier{
		cfg:                cfg,
		transactionService: transactionService,
		logger:             logger,
		httpClient:         &http.Client{Timeout: 10 * time.Second},
	}
}

type noopInsightsNotifier struct{}

func (noopInsightsNotifier) TransactionCreated(*model.Transaction) {}

type eventsInsightsNotifier struct {
	cfg                config.InsightsConfig
	transactionService TransactionsService
	logger             *slog.Logger
	httpClient         *http.Client
}

func (n *eventsInsightsNotifier) TransactionCreated(tx *model.Transaction) {
	go n.notify(tx)
}

func (n *eventsInsightsNotifier) notify(tx *model.Transaction) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if !n.isSignificant(ctx, tx) {
		return
	}

	if n.hasPendingRebuild(ctx) {
		n.logger.Debug("insight rebuild already pending, skipping enqueue", "transaction_id", tx.ID)
		return
	}

	if err := n.enqueueRebuild(ctx, tx); err != nil {
		n.logger.Error("failed to enqueue insight rebuild", "error", err, "transaction_id", tx.ID)
		return
	}
	n.logger.Info("enqueued insight rebuild", "transaction_id", tx.ID, "amount", tx.Amount)
}

// isSignificant decides whether the new expense should invalidate the cached
// insight: either its absolute amount crosses the configured threshold, or it
// alone represents a meaningful share of the current month's expenses.
func (n *eventsInsightsNotifier) isSignificant(ctx context.Context, tx *model.Transaction) bool {
	if tx.Type != string(model.Expense) {
		return false
	}
	if n.cfg.SignificantAmount > 0 && tx.Amount >= n.cfg.SignificantAmount {
		return true
	}
	if n.cfg.SignificantMonthPercent <= 0 {
		return false
	}

	percentages, err := n.transactionService.GetTransactionMonthlyPercentages(ctx, tx)
	if err != nil {
		n.logger.Warn("failed to compute transaction significance", "error", err, "transaction_id", tx.ID)
		return false
	}
	return percentages.TotalMonthPercent != nil &&
		*percentages.TotalMonthPercent >= n.cfg.SignificantMonthPercent
}

// hasPendingRebuild checks (best-effort) whether a rebuild job is already
// waiting in the queue, so a burst of significant transactions triggers a
// single regeneration instead of one per transaction.
func (n *eventsInsightsNotifier) hasPendingRebuild(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.cfg.EventsBaseURL+"/api/events?status=pending", nil)
	if err != nil {
		return false
	}
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	var events []struct {
		JobType string `json:"job_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return false
	}
	for _, event := range events {
		if event.JobType == RebuildInsightsJobType {
			return true
		}
	}
	return false
}

func (n *eventsInsightsNotifier) enqueueRebuild(ctx context.Context, tx *model.Transaction) error {
	payload, err := json.Marshal(map[string]interface{}{
		"reason":         "significant_transaction_created",
		"transaction_id": tx.ID,
		"amount":         tx.Amount,
		"category_id":    tx.CategoryID,
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	body, err := json.Marshal(map[string]string{
		"job_type":     RebuildInsightsJobType,
		"payload":      string(payload),
		"callback_url": n.cfg.RebuildCallbackURL,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.EventsBaseURL+"/api/events", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s/api/events failed: %w", n.cfg.EventsBaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Include a bounded snippet of the response body so a rejected enqueue
		// is debuggable instead of a bare status code.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if body := strings.TrimSpace(string(snippet)); body != "" {
			return fmt.Errorf("POST %s/api/events returned %d: %s", n.cfg.EventsBaseURL, resp.StatusCode, body)
		}
		return fmt.Errorf("POST %s/api/events returned %d", n.cfg.EventsBaseURL, resp.StatusCode)
	}
	return nil
}
