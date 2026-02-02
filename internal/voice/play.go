package voice

import (
	"discordAudio/internal/discordUtils"
	"discordAudio/internal/radio"
	"discordAudio/internal/stream"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func PlayRadio(s *discordgo.Session, m *discordgo.MessageCreate) error {
	idxStr := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(m.Content), "!play "))
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		_, err := s.ChannelMessageSend(m.ChannelID, "Неверный номер")
		if err != nil {
			return err
		}
		return nil
	}

	user := m.Author.ID
	stations, ok := radio.RecentSearch[user]
	if !ok || idx <= 0 || idx > len(stations) {
		_, err := s.ChannelMessageSend(m.ChannelID, "Не найдено для этого номера")
		if err != nil {
			return err
		}
		return nil
	}

	radioURL := stations[idx-1].StreamURL
	vc, found := discordUtils.FindVoiceConnection(s, m.GuildID)
	if !found {
		err := JoinVoice(s, m)
		if err != nil {
			return err
		}
		vc, found = discordUtils.FindVoiceConnection(s, m.GuildID)
	}

	// отправляем сигнал предыдущему потоку, если есть
	stream.StopChan()

	go stream.StreamRadio(vc, radioURL)
	err = vc.Speaking(true)
	if err != nil {
		return err
	}

	_, err = s.ChannelMessageSend(m.ChannelID, "🎧 Стрим: "+stations[idx-1].Name)
	if err != nil {
		return err
	}
	return nil
}
