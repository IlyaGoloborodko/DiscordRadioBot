package voice

import (
	"discordAudio/internal/discordUtils"
	"discordAudio/internal/stream"

	"github.com/bwmarrin/discordgo"
)

func StopRadio(s *discordgo.Session, m *discordgo.MessageCreate) error {
	vc, found := discordUtils.FindVoiceConnection(s, m.GuildID)
	if !found {
		return nil
	}

	// Останавливаем передачу аудио
	// Оповещаем Discord, что бот больше не говорит
	stream.StopCurrentStream()
	err := vc.Speaking(false)
	if err != nil {
		return err
	}
	return nil
}
