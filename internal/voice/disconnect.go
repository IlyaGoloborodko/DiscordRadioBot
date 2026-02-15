package voice

import (
	"discordAudio/internal/discordUtils"

	"github.com/bwmarrin/discordgo"
)

func DisconnectChannel(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	vc, found := discordUtils.FindVoiceConnection(s, i.GuildID)
	if !found {
		return nil
	}
	err := vc.Disconnect()
	if err != nil {
		return err
	}
	return nil
}
