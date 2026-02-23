package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrRuleNotFound = errors.New("event rule not found")
)

type RuleService struct {
	ruleRepo  repository.EventRuleRepository
	pixelRepo repository.PixelRepository
}

func NewRuleService(ruleRepo repository.EventRuleRepository, pixelRepo repository.PixelRepository) *RuleService {
	return &RuleService{ruleRepo: ruleRepo, pixelRepo: pixelRepo}
}

type CreateRuleInput struct {
	PageURL     string          `json:"page_url" validate:"required"`
	EventName   string          `json:"event_name" validate:"required"`
	TriggerType string          `json:"trigger_type" validate:"required,oneof=click pageview scroll form_submit custom"`
	CSSSelector string          `json:"css_selector,omitempty"`
	XPath       string          `json:"xpath,omitempty"`
	ElementText string          `json:"element_text,omitempty"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	FireOnce    bool            `json:"fire_once"`
	DelayMs     int             `json:"delay_ms"`
}

type UpdateRuleInput struct {
	PageURL     *string         `json:"page_url,omitempty"`
	EventName   *string         `json:"event_name,omitempty"`
	TriggerType *string         `json:"trigger_type,omitempty"`
	CSSSelector *string         `json:"css_selector,omitempty"`
	XPath       *string         `json:"xpath,omitempty"`
	ElementText *string         `json:"element_text,omitempty"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	FireOnce    *bool           `json:"fire_once,omitempty"`
	DelayMs     *int            `json:"delay_ms,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
}

func (s *RuleService) Create(ctx context.Context, customerID, pixelID string, input CreateRuleInput) (*domain.EventRule, error) {
	pixel, err := s.pixelRepo.GetByID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("get pixel: %w", err)
	}
	if pixel == nil || pixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	rule := &domain.EventRule{
		PixelID:     pixelID,
		PageURL:     input.PageURL,
		EventName:   input.EventName,
		TriggerType: input.TriggerType,
		CSSSelector: input.CSSSelector,
		XPath:       input.XPath,
		ElementText: input.ElementText,
		Conditions:  input.Conditions,
		Parameters:  input.Parameters,
		FireOnce:    input.FireOnce,
		DelayMs:     input.DelayMs,
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) ListByPixelID(ctx context.Context, customerID, pixelID string) ([]*domain.EventRule, error) {
	pixel, err := s.pixelRepo.GetByID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("get pixel: %w", err)
	}
	if pixel == nil || pixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	rules, err := s.ruleRepo.ListByPixelID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	if rules == nil {
		rules = []*domain.EventRule{}
	}
	return rules, nil
}

func (s *RuleService) Update(ctx context.Context, customerID, ruleID string, input UpdateRuleInput) (*domain.EventRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}
	if rule == nil {
		return nil, ErrRuleNotFound
	}

	// Verify ownership via pixel
	pixel, err := s.pixelRepo.GetByID(ctx, rule.PixelID)
	if err != nil || pixel == nil || pixel.CustomerID != customerID {
		return nil, ErrPixelNotOwned
	}

	if input.PageURL != nil {
		rule.PageURL = *input.PageURL
	}
	if input.EventName != nil {
		rule.EventName = *input.EventName
	}
	if input.TriggerType != nil {
		rule.TriggerType = *input.TriggerType
	}
	if input.CSSSelector != nil {
		rule.CSSSelector = *input.CSSSelector
	}
	if input.XPath != nil {
		rule.XPath = *input.XPath
	}
	if input.ElementText != nil {
		rule.ElementText = *input.ElementText
	}
	if input.Conditions != nil {
		rule.Conditions = input.Conditions
	}
	if input.Parameters != nil {
		rule.Parameters = input.Parameters
	}
	if input.FireOnce != nil {
		rule.FireOnce = *input.FireOnce
	}
	if input.DelayMs != nil {
		rule.DelayMs = *input.DelayMs
	}
	if input.IsActive != nil {
		rule.IsActive = *input.IsActive
	}

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Delete(ctx context.Context, customerID, ruleID string) error {
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("get rule: %w", err)
	}
	if rule == nil {
		return ErrRuleNotFound
	}

	pixel, err := s.pixelRepo.GetByID(ctx, rule.PixelID)
	if err != nil || pixel == nil || pixel.CustomerID != customerID {
		return ErrPixelNotOwned
	}

	return s.ruleRepo.Delete(ctx, ruleID)
}

func (s *RuleService) ListActiveByPixelID(ctx context.Context, pixelID string) ([]*domain.EventRule, error) {
	rules, err := s.ruleRepo.ListActiveByPixelID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("list active rules: %w", err)
	}
	if rules == nil {
		rules = []*domain.EventRule{}
	}
	return rules, nil
}
