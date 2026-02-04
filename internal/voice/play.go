package voice

import (
	"discordAudio/internal/discordUtils"
	"discordAudio/internal/radio"
	"discordAudio/internal/stream"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

//func PlayRadio(s *discordgo.Session, m *discordgo.MessageCreate) error {
//	idxStr := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(m.Content), "!play "))
//	idx, err := strconv.Atoi(idxStr)
//	if err != nil {
//		_, err := s.ChannelMessageSend(m.ChannelID, "Неверный номер")
//		if err != nil {
//			return err
//		}
//		return nil
//	}
//
//	user := m.Author.ID
//	stations, ok := radio.RecentSearch[user]
//	if !ok || idx <= 0 || idx > len(stations) {
//		_, err := s.ChannelMessageSend(m.ChannelID, "Не найдено для этого номера")
//		if err != nil {
//			return err
//		}
//		return nil
//	}
//
//	radioURL := stations[idx-1].StreamURL
//	vc, found := discordUtils.FindVoiceConnection(s, m.GuildID)
//	if !found {
//		err := JoinVoice(s, m)
//		if err != nil {
//			return err
//		}
//		vc, found = discordUtils.FindVoiceConnection(s, m.GuildID)
//	}
//
//	// отправляем сигнал предыдущему потоку, если есть
//	stream.StopChan()
//
//	go func() {
//		err := stream.StartStreaming(vc, radioURL)
//		if err != nil {
//			log.Fatalf("error playing radio: %v", err)
//		}
//
//	}()
//	err = vc.Speaking(true)
//	if err != nil {
//		return err
//	}
//
//	_, err = s.ChannelMessageSend(m.ChannelID, "🎧 Стрим: "+stations[idx-1].Name)
//	if err != nil {
//		return err
//	}
//	return nil
//}

func PlayRadio(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	selectedUUID := i.ApplicationCommandData().Options[0].StringValue()

	var station *radio.Station
	for _, st := range radio.AllStations {
		if st.StationUUID == selectedUUID {
			station = &st
			break
		}
	}
	if station == nil {
		return nil
	}

	radioURL := station.StreamURL

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return err
	}

	// Пытаемся найти существующее соединение
	vc, found := discordUtils.FindVoiceConnection(s, i.GuildID)
	if !found || vc == nil {
		// Если нет — подключаемся
		vc, err = JoinVoice(s, i)
		if err != nil {
			// Тут можно ответить пользователю
			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Сначала зайди в голосовой канал!",
			})
			return nil
		}

		// Ждём, чтобы соединение реально установилось
		time.Sleep(time.Second)
	}

	time.Sleep(250 * time.Millisecond)

	// Останавливаем предыдущий стрим (если был)
	stream.StopChan()

	// Запускаем стрим в отдельной горутине
	go func() {
		if err := stream.StartStreaming(vc, radioURL); err != nil {
			log.Println("Error streaming:", err)
		}
	}()

	// Включаем speaking
	if err := vc.Speaking(true); err != nil {
		log.Println("Error setting speaking:", err)
	}

	// Отправляем сообщение пользователю
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "🎧 Стрим: " + station.Name + " " + station.Country,
	})
	return err
}
