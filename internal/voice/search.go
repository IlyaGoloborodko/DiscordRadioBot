package voice

import (
	"discordAudio/internal/radio"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func searchRadio(query string) ([]radio.Station, error) {
	content := strings.ToLower(query)

	keyword := strings.TrimSpace(strings.TrimPrefix(content, "!search "))
	matches := searchStations(keyword)

	if len(matches) == 0 {
		return nil, nil
	}
	maxCount := 10
	if len(matches) < maxCount {
		maxCount = len(matches)
	}
	return matches, nil
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
func Search(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	query := i.ApplicationCommandData().Options[0].StringValue()

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	foundRadios, err := searchRadio(query)
	if err != nil {
	}
	for _, r := range foundRadios {
		displayName := r.Name
		if r.Country != "" {
			displayName += " (" + r.Country + ")"
		}

		if len(displayName) > 100 {
			displayName = displayName[:100]
		}

		if len(displayName) == 0 {
			displayName = "Unknown Station"
		}

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  displayName,
			Value: r.StationUUID,
		})
		if len(choices) >= 25 {
			break
		}
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult, // 🔑 тип для автодополнения
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Fatalf("Failed to interact interaction: %v", err)
	}

	return nil

}
