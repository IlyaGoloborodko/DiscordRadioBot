package voice

import (
	"discordAudio/internal/radio"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func Search(s *discordgo.Session, m *discordgo.MessageCreate) error {
	content := strings.ToLower(m.Content)

	keyword := strings.TrimSpace(strings.TrimPrefix(content, "!find "))
	matches := searchStations(keyword)

	if len(matches) == 0 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Ничего не найдено по запросу “"+keyword+"”")
		if err != nil {
			return err
		}
		return nil
	}

	maxCount := 10
	if len(matches) < maxCount {
		maxCount = len(matches)
	}

	msg := "Найденные станции:\n"
	for i := 0; i < maxCount; i++ {
		st := matches[i]
		msg += fmt.Sprintf("%d) %s — %s (%s)\n", i+1, st.Name, st.Country, st.StreamURL)
	}
	msg += "\nИспользуй `!play <номер>` чтобы включить станцию."

	_, err := s.ChannelMessageSend(m.ChannelID, msg)
	if err != nil {
		return err
	}
	radio.RecentSearch[m.Author.ID] = matches[:maxCount]
	return nil
}

func searchStations(term string) []radio.Station {
	term = strings.ToLower(term)
	res := make([]radio.Station, 0)
	for _, st := range radio.AllStations {
		if strings.Contains(strings.ToLower(st.Name), term) ||
			strings.Contains(strings.ToLower(st.Country), term) {
			res = append(res, st)
		}
	}
	return res
}
