package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"death-level-tracker/internal/adapters/discord/formatting"
	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
	"death-level-tracker/internal/core/services"

	"github.com/bwmarrin/discordgo"
)

type mockStorage struct {
	saveGuildWorldFunc        func(ctx context.Context, guildID, world string) error
	deleteGuildConfigFunc     func(ctx context.Context, guildID string) error
	getGuildConfigFunc        func(ctx context.Context, guildID string) (*domain.GuildConfig, error)
	addGuildToConfigFunc      func(ctx context.Context, guildID, tibiaGuild string) error
	removeGuildFromConfigFunc func(ctx context.Context, guildID, tibiaGuild string) error
}

func (m *mockStorage) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	if m.saveGuildWorldFunc != nil {
		return m.saveGuildWorldFunc(ctx, guildID, world)
	}
	return nil
}

func (m *mockStorage) DeleteGuildConfig(ctx context.Context, guildID string) error {
	if m.deleteGuildConfigFunc != nil {
		return m.deleteGuildConfigFunc(ctx, guildID)
	}
	return nil
}

func (m *mockStorage) GetAllGuildConfigs(ctx context.Context) ([]domain.GuildConfig, error) {
	return nil, nil
}

func (m *mockStorage) AddGuildToConfig(ctx context.Context, guildID, tibiaGuild string) error {
	if m.addGuildToConfigFunc != nil {
		return m.addGuildToConfigFunc(ctx, guildID, tibiaGuild)
	}
	return nil
}

func (m *mockStorage) RemoveGuildFromConfig(ctx context.Context, guildID, tibiaGuild string) error {
	if m.removeGuildFromConfigFunc != nil {
		return m.removeGuildFromConfigFunc(ctx, guildID, tibiaGuild)
	}
	return nil
}

func (m *mockStorage) GetGuildConfig(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
	if m.getGuildConfigFunc != nil {
		return m.getGuildConfigFunc(ctx, guildID)
	}
	return nil, nil
}

func (m *mockStorage) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	return nil, nil
}

func (m *mockStorage) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	return nil
}

func (m *mockStorage) BatchTouchPlayers(ctx context.Context, names []string) error {
	return nil
}

func (m *mockStorage) DeleteOldPlayers(ctx context.Context, world string, threshold time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockStorage) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error) {
	return nil, nil
}

func (m *mockStorage) Close() {}

type mockDiscordSession struct {
	guildChannelsFunc      func(guildID string) ([]*discordgo.Channel, error)
	guildChannelCreateFunc func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error)
	interactionRespondFunc func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse) error

	lastInteractionResponse *discordgo.InteractionResponse
}

func (m *mockDiscordSession) GuildChannels(guildID string, opts ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	if m.guildChannelsFunc != nil {
		return m.guildChannelsFunc(guildID)
	}
	return []*discordgo.Channel{}, nil
}

func (m *mockDiscordSession) GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, opts ...discordgo.RequestOption) (*discordgo.Channel, error) {
	if m.guildChannelCreateFunc != nil {
		return m.guildChannelCreateFunc(guildID, name, ctype)
	}
	return &discordgo.Channel{ID: "mock-id", Name: name, Type: ctype}, nil
}

func (m *mockDiscordSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
	m.lastInteractionResponse = resp
	if m.interactionRespondFunc != nil {
		return m.interactionRespondFunc(interaction, resp)
	}
	return nil
}

func newTestHandler(storage *mockStorage) *BotHandler {
	return &BotHandler{
		Config: &config.Config{
			DiscordChannelDeath: "death-tracker",
			DiscordChannelLevel: "level-tracker",
		},
		Service: services.NewConfigurationService(storage),
	}
}

func makeCommandInteraction(guildID, optName, optValue string) *discordgo.InteractionCreate {
	var opts []*discordgo.ApplicationCommandInteractionDataOption
	if optName != "" {
		opts = []*discordgo.ApplicationCommandInteractionDataOption{
			{Name: optName, Type: discordgo.ApplicationCommandOptionString, Value: optValue},
		}
	}
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: guildID,
			Data:    discordgo.ApplicationCommandInteractionData{Options: opts},
		},
	}
}

func sessionWithChannels(channels ...*discordgo.Channel) *mockDiscordSession {
	return &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			return channels, nil
		},
	}
}

func TestTrackWorld_Success(t *testing.T) {
	var savedWorld string
	storage := &mockStorage{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			savedWorld = world
			return nil
		},
	}

	session := sessionWithChannels(
		&discordgo.Channel{ID: "1", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
		&discordgo.Channel{ID: "2", Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
	)

	handler := newTestHandler(storage)
	handler.TrackWorld(session, makeCommandInteraction("guild-1", "name", "antica"))

	if savedWorld != "Antica" {
		t.Errorf("expected world 'Antica', got '%s'", savedWorld)
	}
	if session.lastInteractionResponse.Data.Flags != 0 {
		t.Error("expected non-ephemeral success message")
	}
}

func TestTrackWorld_MissingWorldName(t *testing.T) {
	session := &mockDiscordSession{}
	handler := newTestHandler(&mockStorage{})

	handler.TrackWorld(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgWorldRequired {
		t.Errorf("expected '%s', got '%s'", formatting.MsgWorldRequired, session.lastInteractionResponse.Data.Content)
	}
	if session.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("expected ephemeral error message")
	}
}

func TestTrackWorld_ChannelCreation(t *testing.T) {
	var created []string
	storage := &mockStorage{}
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{}, nil
		},
		guildChannelCreateFunc: func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error) {
			created = append(created, name)
			return &discordgo.Channel{ID: "new-" + name, Name: name, Type: ctype}, nil
		},
	}

	handler := newTestHandler(storage)
	handler.TrackWorld(session, makeCommandInteraction("guild-1", "name", "secura"))

	if len(created) != 2 {
		t.Fatalf("expected 2 channels created, got %d", len(created))
	}
	if created[0] != "death-tracker" || created[1] != "level-tracker" {
		t.Errorf("expected [death-tracker, level-tracker], got %v", created)
	}
}

func TestTrackWorld_ChannelError(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			return nil, errors.New("api error")
		},
	}

	handler := newTestHandler(&mockStorage{})
	handler.TrackWorld(session, makeCommandInteraction("guild-1", "name", "antica"))

	expected := formatting.MsgChannelError("death-tracker")
	if session.lastInteractionResponse.Data.Content != expected {
		t.Errorf("expected '%s', got '%s'", expected, session.lastInteractionResponse.Data.Content)
	}
}

func TestTrackWorld_StorageError(t *testing.T) {
	storage := &mockStorage{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			return errors.New("db error")
		},
	}

	session := sessionWithChannels(
		&discordgo.Channel{Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
		&discordgo.Channel{Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
	)

	handler := newTestHandler(storage)
	handler.TrackWorld(session, makeCommandInteraction("guild-1", "name", "antica"))

	if session.lastInteractionResponse.Data.Content != formatting.MsgSaveError {
		t.Errorf("expected '%s', got '%s'", formatting.MsgSaveError, session.lastInteractionResponse.Data.Content)
	}
}

func TestTrackWorld_WorldNameFormatting(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"antica", "Antica"},
		{"SECURA", "Secura"},
		{"beLaBonA", "Belabona"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var saved string
			storage := &mockStorage{
				saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
					saved = world
					return nil
				},
			}

			session := sessionWithChannels(
				&discordgo.Channel{Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
				&discordgo.Channel{Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
			)

			handler := newTestHandler(storage)
			handler.TrackWorld(session, makeCommandInteraction("guild-1", "name", tt.input))

			if saved != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, saved)
			}
		})
	}
}

func TestStopTracking_Success(t *testing.T) {
	var deleted string
	storage := &mockStorage{
		deleteGuildConfigFunc: func(ctx context.Context, guildID string) error {
			deleted = guildID
			return nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.StopTracking(session, makeCommandInteraction("guild-123", "", ""))

	if deleted != "guild-123" {
		t.Errorf("expected 'guild-123', got '%s'", deleted)
	}
	if session.lastInteractionResponse.Data.Content != formatting.MsgStopSuccess {
		t.Errorf("expected '%s', got '%s'", formatting.MsgStopSuccess, session.lastInteractionResponse.Data.Content)
	}
}

func TestStopTracking_Error(t *testing.T) {
	storage := &mockStorage{
		deleteGuildConfigFunc: func(ctx context.Context, guildID string) error {
			return errors.New("db error")
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.StopTracking(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgStopError {
		t.Errorf("expected '%s', got '%s'", formatting.MsgStopError, session.lastInteractionResponse.Data.Content)
	}
}

func TestAddGuild_Success(t *testing.T) {
	var added string
	storage := &mockStorage{
		addGuildToConfigFunc: func(ctx context.Context, guildID, tibiaGuild string) error {
			added = tibiaGuild
			return nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.AddGuild(session, makeCommandInteraction("guild-1", "name", "Red Rose"))

	if added != "Red Rose" {
		t.Errorf("expected 'Red Rose', got '%s'", added)
	}

	expected := formatting.MsgGuildAdded("Red Rose")
	if session.lastInteractionResponse.Data.Content != expected {
		t.Errorf("expected '%s', got '%s'", expected, session.lastInteractionResponse.Data.Content)
	}
}

func TestAddGuild_MissingName(t *testing.T) {
	session := &mockDiscordSession{}
	handler := newTestHandler(&mockStorage{})
	handler.AddGuild(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgGuildNameRequired {
		t.Errorf("expected '%s'", formatting.MsgGuildNameRequired)
	}
}

func TestAddGuild_Error(t *testing.T) {
	storage := &mockStorage{
		addGuildToConfigFunc: func(ctx context.Context, guildID, tibiaGuild string) error {
			return errors.New("db error")
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.AddGuild(session, makeCommandInteraction("guild-1", "name", "Test"))

	if session.lastInteractionResponse.Data.Content != formatting.MsgSaveError {
		t.Errorf("expected '%s'", formatting.MsgSaveError)
	}
}

func TestUnsetGuild_Success(t *testing.T) {
	var removed string
	storage := &mockStorage{
		removeGuildFromConfigFunc: func(ctx context.Context, guildID, tibiaGuild string) error {
			removed = tibiaGuild
			return nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.UnsetGuild(session, makeCommandInteraction("guild-1", "name", "Red Rose"))

	if removed != "Red Rose" {
		t.Errorf("expected 'Red Rose', got '%s'", removed)
	}

	expected := formatting.MsgGuildRemoved("Red Rose")
	if session.lastInteractionResponse.Data.Content != expected {
		t.Errorf("expected '%s', got '%s'", expected, session.lastInteractionResponse.Data.Content)
	}
}

func TestUnsetGuild_MissingName(t *testing.T) {
	session := &mockDiscordSession{}
	handler := newTestHandler(&mockStorage{})
	handler.UnsetGuild(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgGuildNameRequired {
		t.Errorf("expected '%s'", formatting.MsgGuildNameRequired)
	}
}

func TestUnsetGuild_Autocomplete(t *testing.T) {
	storage := &mockStorage{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return &domain.GuildConfig{TibiaGuilds: []string{"Red Rose", "Blue Army"}}, nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommandAutocomplete,
			GuildID: "guild-1",
			Data: discordgo.ApplicationCommandInteractionData{
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "red", Focused: true},
				},
			},
		},
	}

	handler.UnsetGuild(session, interaction)

	if session.lastInteractionResponse.Type != discordgo.InteractionApplicationCommandAutocompleteResult {
		t.Error("expected autocomplete response type")
	}
	if len(session.lastInteractionResponse.Data.Choices) != 1 {
		t.Errorf("expected 1 choice matching 'red', got %d", len(session.lastInteractionResponse.Data.Choices))
	}
}

func TestListGuilds_WithGuilds(t *testing.T) {
	storage := &mockStorage{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return &domain.GuildConfig{TibiaGuilds: []string{"Red Rose", "Blue Army"}}, nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.ListGuilds(session, makeCommandInteraction("guild-1", "", ""))

	expected := formatting.MsgGuildsList([]string{"Red Rose", "Blue Army"})
	if session.lastInteractionResponse.Data.Content != expected {
		t.Errorf("expected '%s', got '%s'", expected, session.lastInteractionResponse.Data.Content)
	}
}

func TestListGuilds_NoGuilds(t *testing.T) {
	storage := &mockStorage{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return &domain.GuildConfig{TibiaGuilds: []string{}}, nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.ListGuilds(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgNoGuildsTracked {
		t.Errorf("expected '%s'", formatting.MsgNoGuildsTracked)
	}
}

func TestListGuilds_NilConfig(t *testing.T) {
	storage := &mockStorage{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return nil, nil
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.ListGuilds(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgNoGuildsTracked {
		t.Errorf("expected '%s'", formatting.MsgNoGuildsTracked)
	}
}

func TestListGuilds_Error(t *testing.T) {
	storage := &mockStorage{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return nil, errors.New("db error")
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.ListGuilds(session, makeCommandInteraction("guild-1", "", ""))

	if session.lastInteractionResponse.Data.Content != formatting.MsgConfigError {
		t.Errorf("expected '%s'", formatting.MsgConfigError)
	}
}

func TestBuildGuildChoices(t *testing.T) {
	t.Run("filters by query", func(t *testing.T) {
		cfg := &domain.GuildConfig{TibiaGuilds: []string{"Red Rose", "Blue Army", "Red Dragons"}}
		choices := buildGuildChoices(cfg, "red")

		if len(choices) != 2 {
			t.Fatalf("expected 2 choices, got %d", len(choices))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		cfg := &domain.GuildConfig{TibiaGuilds: []string{"Red Rose"}}
		choices := buildGuildChoices(cfg, "RED")

		if len(choices) != 1 {
			t.Error("expected case-insensitive match")
		}
	})

	t.Run("empty query returns all", func(t *testing.T) {
		cfg := &domain.GuildConfig{TibiaGuilds: []string{"A", "B", "C"}}
		choices := buildGuildChoices(cfg, "")

		if len(choices) != 3 {
			t.Errorf("expected 3, got %d", len(choices))
		}
	})

	t.Run("limits to 25", func(t *testing.T) {
		guilds := make([]string, 30)
		for i := range guilds {
			guilds[i] = "Guild"
		}
		cfg := &domain.GuildConfig{TibiaGuilds: guilds}
		choices := buildGuildChoices(cfg, "")

		if len(choices) != 25 {
			t.Errorf("expected 25 max, got %d", len(choices))
		}
	})

	t.Run("nil config returns nil", func(t *testing.T) {
		choices := buildGuildChoices(nil, "test")
		if choices != nil {
			t.Error("expected nil")
		}
	})
}

func TestReadyHandler(t *testing.T) {
	session := &discordgo.Session{State: discordgo.NewState()}
	session.State.User = &discordgo.User{Username: "TestBot", Discriminator: "1234"}

	ReadyHandler(session, &discordgo.Ready{})
}

func TestTrackWorld_LevelChannelError(t *testing.T) {
	callCount := 0
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			callCount++
			if callCount == 1 {
				return []*discordgo.Channel{
					{Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
				}, nil
			}
			return nil, errors.New("level channel error")
		},
	}

	handler := newTestHandler(&mockStorage{})
	handler.TrackWorld(session, makeCommandInteraction("guild-1", "name", "antica"))

	expected := formatting.MsgChannelError("level-tracker")
	if session.lastInteractionResponse.Data.Content != expected {
		t.Errorf("expected '%s', got '%s'", expected, session.lastInteractionResponse.Data.Content)
	}
}

func TestUnsetGuild_Error(t *testing.T) {
	storage := &mockStorage{
		removeGuildFromConfigFunc: func(ctx context.Context, guildID, tibiaGuild string) error {
			return errors.New("db error")
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)
	handler.UnsetGuild(session, makeCommandInteraction("guild-1", "name", "Test Guild"))

	if session.lastInteractionResponse.Data.Content != formatting.MsgSaveError {
		t.Errorf("expected '%s'", formatting.MsgSaveError)
	}
}

func TestUnsetGuild_AutocompleteError(t *testing.T) {
	storage := &mockStorage{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return nil, errors.New("db error")
		},
	}

	session := &mockDiscordSession{}
	handler := newTestHandler(storage)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommandAutocomplete,
			GuildID: "guild-1",
			Data: discordgo.ApplicationCommandInteractionData{
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "test", Focused: true},
				},
			},
		},
	}

	handler.UnsetGuild(session, interaction)

	if session.lastInteractionResponse != nil {
		t.Error("expected no response when GetGuildConfig fails in autocomplete")
	}
}
