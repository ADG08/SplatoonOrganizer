package guildconfig

// GuildConfig holds optional channel and role IDs for a guild.
// Empty strings mean use default channel / no role ping.
type GuildConfig struct {
	ChannelID string
	RoleID    string
}
