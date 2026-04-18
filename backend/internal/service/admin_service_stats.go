package service

import (
	"context"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func (s *AdminService) GetPlatformOverview(ctx context.Context) (*domain.PlatformStats, error) {
	if cached, ok := s.statsCache.get(); ok {
		return cached, nil
	}
	stats, err := s.adminRepo.GetPlatformStats(ctx)
	if err != nil {
		return nil, err
	}
	s.statsCache.set(stats)
	return stats, nil
}

func (s *AdminService) GetRevenueChart(ctx context.Context, days int) ([]*domain.RevenueChartPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	return s.adminRepo.GetRevenueChart(ctx, days)
}

func (s *AdminService) GetGrowthChart(ctx context.Context, days int) ([]*domain.GrowthChartPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	return s.adminRepo.GetGrowthChart(ctx, days)
}
