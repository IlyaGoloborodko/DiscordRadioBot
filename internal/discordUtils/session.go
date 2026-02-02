package discordUtils

import "github.com/bwmarrin/discordgo"

func FindVoiceConnection(s *discordgo.Session, guildID string) (*discordgo.VoiceConnection, bool) {
	for _, conn := range s.VoiceConnections {
		if conn.GuildID == guildID {
			return conn, true
		}
	}
	return nil, false
}
