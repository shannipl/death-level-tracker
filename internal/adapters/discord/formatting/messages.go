package formatting

import "fmt"

const (
	MsgAdminRequired     = "You need Administrator permissions to use this command."
	MsgWorldRequired     = "World name is required."
	MsgGuildNameRequired = "Guild name is required."
	MsgSaveError         = "Failed to save configuration."
	MsgStopError         = "Failed to stop tracking."
	MsgStopSuccess       = "Tracking stopped. Configuration removed."
	MsgConfigError       = "Failed to retrieve configuration."
	MsgNoGuildsTracked   = "No guilds are currently being tracked (all players will be tracked)."
)

func MsgDeath(name, timeStr, reason string) string {
	return fmt.Sprintf("%s - %s - %s", name, timeStr, reason)
}

func MsgLevelUp(name string, oldLevel, newLevel int) string {
	return fmt.Sprintf("%s advanced from level %d to %d", name, oldLevel, newLevel)
}

func MsgChannelError(channelName string) string {
	return fmt.Sprintf("Failed to create or find #%s channel.", channelName)
}

func MsgTrackSuccess(world, deathChan, levelChan string) string {
	return fmt.Sprintf("Tracking world **%s** configured! Notifications will appear in #%s and #%s.", world, deathChan, levelChan)
}

func MsgGuildAdded(name string) string {
	return fmt.Sprintf("Added guild '%s' to tracking list.", name)
}

func MsgGuildRemoved(name string) string {
	return fmt.Sprintf("Removed guild '%s' from tracking list.", name)
}

func MsgGuildsList(guilds []string) string {
	msg := "Tracking specific guilds:\n"
	for _, g := range guilds {
		msg += "- " + g + "\n"
	}
	return msg
}
