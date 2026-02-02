package voice

import (
	"github.com/bwmarrin/discordgo"
)

func JoinVoice(s *discordgo.Session, m *discordgo.MessageCreate) error {
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		return err
	}

	var vcID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			vcID = vs.ChannelID
			break
		}
	}

	if vcID == "" {
		return nil
	}

	_, err = s.ChannelVoiceJoin(m.GuildID, vcID, false, true)
	if err != nil {
		return err
	}
	return nil
}
