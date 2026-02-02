package voice

import (
	"discordAudio/internal/discordUtils"

	"github.com/bwmarrin/discordgo"
)

func DisconnectChannel(s *discordgo.Session, m *discordgo.MessageCreate) error {
	vc, found := discordUtils.FindVoiceConnection(s, m.GuildID)
	if !found {
		return nil
	}
	// Закрываем голосовое соединение
	err := vc.Disconnect()
	if err != nil {
		return err
	}
	return nil
}
