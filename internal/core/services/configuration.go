package services

import (
	"context"
	"strings"

	"death-level-tracker/internal/core/domain"
	"death-level-tracker/internal/core/ports"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ConfigurationService struct {
	repo ports.Repository
}

func NewConfigurationService(repo ports.Repository) *ConfigurationService {
	return &ConfigurationService{repo: repo}
}

func (s *ConfigurationService) SetWorld(ctx context.Context, guildID, worldName string) (string, error) {
	formattedWorld := cases.Title(language.English).String(strings.ToLower(worldName))
	err := s.repo.SaveGuildWorld(ctx, guildID, formattedWorld)
	return formattedWorld, err
}

func (s *ConfigurationService) StopTracking(ctx context.Context, guildID string) error {
	return s.repo.DeleteGuildConfig(ctx, guildID)
}

func (s *ConfigurationService) AddGuildToTrack(ctx context.Context, guildID, tibiaGuildName string) error {
	return s.repo.AddGuildToConfig(ctx, guildID, tibiaGuildName)
}

func (s *ConfigurationService) RemoveGuildFromTrack(ctx context.Context, guildID, tibiaGuildName string) error {
	return s.repo.RemoveGuildFromConfig(ctx, guildID, tibiaGuildName)
}

func (s *ConfigurationService) GetGuildConfig(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
	return s.repo.GetGuildConfig(ctx, guildID)
}
