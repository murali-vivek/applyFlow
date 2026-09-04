package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/applyflow/applyflow/internal/queue"
	"github.com/applyflow/applyflow/internal/repository"
)

type SchedulerService struct {
	outreach  *repository.OutreachRepository
	campaigns *repository.CampaignRepository
	queue     *queue.SQSClient
	interval  time.Duration
	staleAfter time.Duration
}

func NewSchedulerService(
	outreach *repository.OutreachRepository,
	campaigns *repository.CampaignRepository,
	queue *queue.SQSClient,
	interval, staleAfter time.Duration,
) *SchedulerService {
	return &SchedulerService{
		outreach: outreach, campaigns: campaigns, queue: queue,
		interval: interval, staleAfter: staleAfter,
	}
}

func (s *SchedulerService) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *SchedulerService) tick(ctx context.Context) {
	recovered, err := s.outreach.RecoverStaleQueued(ctx, s.staleAfter)
	if err != nil {
		slog.Error("recover stale queued", "error", err)
	} else if recovered > 0 {
		slog.Info("recovered stale queued outreach", "count", recovered)
	}

	ids, err := s.outreach.ClaimDue(ctx, 50)
	if err != nil {
		slog.Error("claim due outreach", "error", err)
		return
	}

	for _, id := range ids {
		outreach, err := s.outreach.GetByID(ctx, id)
		if err != nil {
			slog.Error("load outreach", "outreach_id", id, "error", err)
			continue
		}

		if err := s.campaigns.MarkRunning(ctx, outreach.CampaignID); err != nil {
			slog.Error("mark campaign running", "campaign_id", outreach.CampaignID, "error", err)
		}

		if err := s.queue.Send(ctx, id); err != nil {
			slog.Error("enqueue outreach", "outreach_id", id, "error", err)
			_ = s.outreach.ResetToPending(ctx, id)
			continue
		}
		slog.Info("outreach enqueued", "outreach_id", id)
	}
}
