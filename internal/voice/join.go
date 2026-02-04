package voice

import (
	"github.com/bwmarrin/discordgo"
)

func JoinVoice(s *discordgo.Session, i *discordgo.InteractionCreate) (*discordgo.VoiceConnection, error) {
	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		return nil, err
	}

	var vcID string
	userID := i.Member.User.ID
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			vcID = vs.ChannelID
			break
		}
	}

	if vcID == "" {
		return nil, nil
	}

	vc, err := s.ChannelVoiceJoin(i.GuildID, vcID, false, true)
	if err != nil {
		return nil, err
	}
	return vc, nil
}
