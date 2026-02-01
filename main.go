package main

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"layeh.com/gopus"
)

var radioURL string

func init() {
	godotenv.Load()
	radioURL = os.Getenv("RADIO_URL")
	if radioURL == "" {
		log.Fatal("RADIO_URL not set")
	}
}

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("error creating Discord session,", err)
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
	}
	log.Println("Bot is up!")

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	content := strings.ToLower(m.Content)

	if strings.HasPrefix(content, "!join") {
		joinVoice(s, m)
	}

	if strings.HasPrefix(content, "!play") {
		playRadio(s, m)
	}
}

func joinVoice(s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Guild not found")
		return
	}

	var vcID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			vcID = vs.ChannelID
			break
		}
	}

	if vcID == "" {
		s.ChannelMessageSend(m.ChannelID, "Join a voice channel first")
		return
	}

	vc, err := s.ChannelVoiceJoin(m.GuildID, vcID, false, true)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to join voice channel")
		return
	}

	vc.Speaking(true)
	s.ChannelMessageSend(m.ChannelID, "Joined voice!")
	log.Println("Connected to voice channel:", vcID)
}

func playRadio(s *discordgo.Session, m *discordgo.MessageCreate) {
	vc, found := findVoiceConnection(s, m.GuildID)
	if !found {
		s.ChannelMessageSend(m.ChannelID, "First use !join")
		return
	}

	go streamRadio(vc, radioURL)
	s.ChannelMessageSend(m.ChannelID, "Streaming radio: "+radioURL)
}

func findVoiceConnection(s *discordgo.Session, guildID string) (*discordgo.VoiceConnection, bool) {
	for _, conn := range s.VoiceConnections {
		if conn.GuildID == guildID {
			return conn, true
		}
	}
	return nil, false
}

func streamRadio(vc *discordgo.VoiceConnection, url string) {
	cmd := exec.Command("ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-i", url,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("FFmpeg stdout error:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Println("FFmpeg start error:", err)
		return
	}

	enc, err := gopus.NewEncoder(48000, 2, gopus.Audio)
	if err != nil {
		log.Println("Opus encoder create error:", err)
		return
	}

	// Buffer for 20ms PCM frames
	pcmBuf := make([]int16, 960*2)

	for {
		err := binary.Read(stdout, binary.LittleEndian, pcmBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("PCM read error:", err)
			break
		}

		opusFrame, err := enc.Encode(pcmBuf, len(pcmBuf)/2, len(pcmBuf)/2)
		if err != nil {
			log.Println("Opus encode error:", err)
			break
		}

		vc.OpusSend <- opusFrame
	}

	cmd.Wait()
	log.Println("Radio stream ended")
}
