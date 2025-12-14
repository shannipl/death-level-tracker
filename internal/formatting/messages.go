package formatting

import "fmt"

const (
	MsgAdminRequired = "You need Administrator permissions to use this command."
	MsgWorldRequired = "World name is required."
	MsgSaveError     = "Failed to save configuration."
	MsgStopError     = "Failed to stop tracking."
	MsgStopSuccess   = "Tracking stopped. Configuration removed."
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
