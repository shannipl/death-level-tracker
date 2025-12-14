package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/formatting"
	"death-level-tracker/internal/storage"

	"github.com/bwmarrin/discordgo"
)

// Mock Storage implementation
type mockStorage struct {
	saveGuildWorldFunc    func(ctx context.Context, guildID, world string) error
	deleteGuildConfigFunc func(ctx context.Context, guildID string) error
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

func (m *mockStorage) GetWorldsMap(ctx context.Context) (map[string][]string, error) {
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

func (m *mockStorage) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]storage.OfflinePlayer, error) {
	return nil, nil
}

func (m *mockStorage) Close() {}

// Mock Discord Session
type mockDiscordSession struct {
	guildChannelsFunc       func(guildID string) ([]*discordgo.Channel, error)
	guildChannelCreateFunc  func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error)
	interactionRespondFunc  func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse) error
	lastInteractionResponse *discordgo.InteractionResponse
	lastInteraction         *discordgo.Interaction
}

func (m *mockDiscordSession) GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	if m.guildChannelsFunc != nil {
		return m.guildChannelsFunc(guildID)
	}
	return []*discordgo.Channel{}, nil
}

func (m *mockDiscordSession) GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	if m.guildChannelCreateFunc != nil {
		return m.guildChannelCreateFunc(guildID, name, ctype)
	}
	return &discordgo.Channel{ID: "mock-channel-id", Name: name, Type: ctype}, nil
}

func (m *mockDiscordSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	m.lastInteraction = interaction
	m.lastInteractionResponse = resp
	if m.interactionRespondFunc != nil {
		return m.interactionRespondFunc(interaction, resp)
	}
	return nil
}

// Test helper to create test interaction
func createTestInteraction(guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: guildID,
			Data: discordgo.ApplicationCommandInteractionData{
				Options: options,
			},
		},
	}
}

func TestTrackWorld_Success(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	var savedGuildID, savedWorld string
	storage := &mockStorage{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			savedGuildID = guildID
			savedWorld = world
			return nil
		},
	}

	mockSession := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			// Return existing channels (both already exist)
			return []*discordgo.Channel{
				{ID: "death-channel-id", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
				{ID: "level-channel-id", Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
	}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{
			Name:  "name",
			Type:  discordgo.ApplicationCommandOptionString,
			Value: "antica",
		},
	}

	interaction := createTestInteraction("test-guild-123", options)

	// Call the actual handler with mocked session
	handler.TrackWorld(mockSession, interaction)

	// Verify storage was called correctly
	if savedGuildID != "test-guild-123" {
		t.Errorf("Expected guildID 'test-guild-123', got '%s'", savedGuildID)
	}
	if savedWorld != "Antica" {
		t.Errorf("Expected world 'Antica' (title case), got '%s'", savedWorld)
	}

	// Verify response was sent
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	expectedMsg := formatting.MsgTrackSuccess("Antica", "death-tracker", "level-tracker")
	if mockSession.lastInteractionResponse.Data.Content != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != 0 {
		t.Error("Expected non-ephemeral message (flags = 0)")
	}
}

func TestTrackWorld_MissingWorldName(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			t.Error("SaveGuildWorld should not be called when world name is missing")
			return nil
		},
	}

	mockSession := &mockDiscordSession{}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{}
	interaction := createTestInteraction("test-guild-123", options)

	handler.TrackWorld(mockSession, interaction)

	// Verify error response was sent
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	if mockSession.lastInteractionResponse.Data.Content != formatting.MsgWorldRequired {
		t.Errorf("Expected message '%s', got '%s'", formatting.MsgWorldRequired, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral message")
	}
}

func TestTrackWorld_ChannelCreation(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			return nil
		},
	}

	channelsCreated := []string{}
	mockSession := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			// No existing channels
			return []*discordgo.Channel{}, nil
		},
		guildChannelCreateFunc: func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error) {
			channelsCreated = append(channelsCreated, name)
			return &discordgo.Channel{ID: "new-" + name, Name: name, Type: ctype}, nil
		},
	}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "secura"},
	}

	interaction := createTestInteraction("test-guild-456", options)
	handler.TrackWorld(mockSession, interaction)

	// Verify both channels were created
	if len(channelsCreated) != 2 {
		t.Fatalf("Expected 2 channels to be created, got %d", len(channelsCreated))
	}
	if channelsCreated[0] != "death-tracker" {
		t.Errorf("Expected first channel 'death-tracker', got '%s'", channelsCreated[0])
	}
	if channelsCreated[1] != "level-tracker" {
		t.Errorf("Expected second channel 'level-tracker', got '%s'", channelsCreated[1])
	}
}

func TestTrackWorld_ChannelError(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{}

	mockSession := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			return nil, errors.New("discord API error")
		},
	}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "antica"},
	}

	interaction := createTestInteraction("test-guild-789", options)
	handler.TrackWorld(mockSession, interaction)

	// Verify error response
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	expectedMsg := formatting.MsgChannelError("death-tracker")
	if mockSession.lastInteractionResponse.Data.Content != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral error message")
	}
}

func TestTrackWorld_LevelChannelError(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{}

	callCount := 0
	mockSession := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			callCount++
			if callCount == 1 {
				// First call for death-tracker - return existing channel
				return []*discordgo.Channel{
					{ID: "death-id", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
				}, nil
			}
			// Second call for level-tracker - return error
			return nil, errors.New("failed to get level channels")
		},
	}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "antica"},
	}

	interaction := createTestInteraction("test-guild-789", options)
	handler.TrackWorld(mockSession, interaction)

	// Verify error response for level channel
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	expectedMsg := formatting.MsgChannelError("level-tracker")
	if mockSession.lastInteractionResponse.Data.Content != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral error message")
	}
}

func TestTrackWorld_ChannelCreationFailure(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{}

	mockSession := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			// No existing channels
			return []*discordgo.Channel{}, nil
		},
		guildChannelCreateFunc: func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error) {
			// Fail to create channel
			return nil, errors.New("permission denied")
		},
	}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "antica"},
	}

	interaction := createTestInteraction("test-guild-999", options)
	handler.TrackWorld(mockSession, interaction)

	// Verify error response
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	expectedMsg := formatting.MsgChannelError("death-tracker")
	if mockSession.lastInteractionResponse.Data.Content != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral error message")
	}
}

func TestTrackWorld_WorldNameFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "lowercase to title case", input: "antica", expected: "Antica"},
		{name: "uppercase to title case", input: "SECURA", expected: "Secura"},
		{name: "mixed case to title case", input: "beLaBonA", expected: "Belabona"},
		{name: "already title case", input: "Pacera", expected: "Pacera"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				DiscordChannelDeath: "death-tracker",
				DiscordChannelLevel: "level-tracker",
			}

			var savedWorld string
			storage := &mockStorage{
				saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
					savedWorld = world
					return nil
				},
			}

			mockSession := &mockDiscordSession{
				guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
					return []*discordgo.Channel{
						{Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
						{Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
					}, nil
				},
			}

			handler := &BotHandler{
				Config: cfg,
				Store:  storage,
			}

			options := []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: tt.input},
			}

			interaction := createTestInteraction("test-guild-123", options)
			handler.TrackWorld(mockSession, interaction)

			if savedWorld != tt.expected {
				t.Errorf("Expected world '%s', got '%s'", tt.expected, savedWorld)
			}
		})
	}
}

func TestTrackWorld_StorageError(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			return errors.New("database connection failed")
		},
	}

	mockSession := &mockDiscordSession{
		guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
				{Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
	}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "antica"},
	}

	interaction := createTestInteraction("test-guild-123", options)
	handler.TrackWorld(mockSession, interaction)

	// Verify error response
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	if mockSession.lastInteractionResponse.Data.Content != formatting.MsgSaveError {
		t.Errorf("Expected message '%s', got '%s'", formatting.MsgSaveError, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral error message")
	}
}

func TestStopTracking_Success(t *testing.T) {
	cfg := &config.Config{}

	var deletedGuildID string
	storage := &mockStorage{
		deleteGuildConfigFunc: func(ctx context.Context, guildID string) error {
			deletedGuildID = guildID
			return nil
		},
	}

	mockSession := &mockDiscordSession{}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	interaction := createTestInteraction("test-guild-456", nil)
	handler.StopTracking(mockSession, interaction)

	// Verify deletion
	if deletedGuildID != "test-guild-456" {
		t.Errorf("Expected guildID 'test-guild-456', got '%s'", deletedGuildID)
	}

	// Verify success response
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	if mockSession.lastInteractionResponse.Data.Content != formatting.MsgStopSuccess {
		t.Errorf("Expected message '%s', got '%s'", formatting.MsgStopSuccess, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != 0 {
		t.Error("Expected non-ephemeral success message")
	}
}

func TestStopTracking_Error(t *testing.T) {
	cfg := &config.Config{}

	storage := &mockStorage{
		deleteGuildConfigFunc: func(ctx context.Context, guildID string) error {
			return errors.New("database error")
		},
	}

	mockSession := &mockDiscordSession{}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	interaction := createTestInteraction("test-guild-456", nil)
	handler.StopTracking(mockSession, interaction)

	// Verify error response
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected interaction response to be sent")
	}
	if mockSession.lastInteractionResponse.Data.Content != formatting.MsgStopError {
		t.Errorf("Expected message '%s', got '%s'", formatting.MsgStopError, mockSession.lastInteractionResponse.Data.Content)
	}
	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral error message")
	}
}

func TestReadyHandler(t *testing.T) {
	// ReadyHandler just logs, so we verify it doesn't panic
	session := &discordgo.Session{
		State: discordgo.NewState(),
	}
	session.State.User = &discordgo.User{
		Username:      "TestBot",
		Discriminator: "1234",
	}

	ready := &discordgo.Ready{}

	// Should not panic
	ReadyHandler(session, ready)
}

func TestBotHandler_Structure(t *testing.T) {
	cfg := &config.Config{
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	storage := &mockStorage{}

	handler := &BotHandler{
		Config: cfg,
		Store:  storage,
	}

	if handler.Config == nil {
		t.Error("Expected Config to be set")
	}

	if handler.Store == nil {
		t.Error("Expected Store to be set")
	}

	if handler.Config.DiscordChannelDeath != "death-tracker" {
		t.Errorf("Expected DiscordChannelDeath 'death-tracker', got '%s'", handler.Config.DiscordChannelDeath)
	}

	if handler.Config.DiscordChannelLevel != "level-tracker" {
		t.Errorf("Expected DiscordChannelLevel 'level-tracker', got '%s'", handler.Config.DiscordChannelLevel)
	}
}
