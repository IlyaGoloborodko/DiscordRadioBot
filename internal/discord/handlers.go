package discord

import (
	"discordAudio/internal/voice"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	content := strings.ToLower(m.Content)

	//if strings.HasPrefix(content, "!join") {
	//	err := voice.JoinVoice(s, m)
	//	if err != nil {
	//		log.Fatalf("error joining voice channel: %v", err)
	//	}
	//	return
	//}

	//if strings.HasPrefix(content, "!play") {
	//	err := voice.PlayRadio(s, m)
	//	if err != nil {
	//		log.Fatalf("error playing radio: %v", err)
	//	}
	//	return
	//}

	if strings.HasPrefix(content, "!stop") {
		err := voice.StopRadio(s, m)
		if err != nil {
			log.Fatalf("error stop radio: %v", err)
		}
		return
	}

	if strings.HasPrefix(content, "!disconnect") {
		err := voice.DisconnectChannel(s, m)
		if err != nil {
			log.Fatalf("error disconnecting channel: %v", err)
		}
	}

	//if strings.HasPrefix(content, "!search ") {
	//	err := voice.Search(s, m)
	//	if err != nil {
	//		log.Fatalf("error searching: %v", err)
	//	}
	//	return
	//}
}
