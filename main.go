package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	_ "strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
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
		s.ChannelMessageSend(m.ChannelID, "Ошибка: не нашёл гильдию")
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
		s.ChannelMessageSend(m.ChannelID, "Сначала зайди в голосовой канал")
		return
	}

	vc, err := s.ChannelVoiceJoin(m.GuildID, vcID, false, true)
	if err != nil {
		_, err := s.ChannelMessageSend(m.ChannelID, "Не получилось подключиться к голосу")
		if err != nil {
			return
		}
		return
	}

	speakingErr := vc.Speaking(true)
	if speakingErr != nil {
		return
	}

	_, chanMessSendErr := s.ChannelMessageSend(m.ChannelID, "Подключился!")
	if chanMessSendErr != nil {
		return
	}
}

func playRadio(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Нужно уже быть в голосовом канале
	vc, found := findVoiceConnection(s, m.GuildID)
	if !found {
		_, err := s.ChannelMessageSend(m.ChannelID, "Сначала !join")
		if err != nil {
			return
		}
		return
	}

	go streamRadio(vc, radioURL)
	_, errorSend := s.ChannelMessageSend(m.ChannelID, "Играть радио: "+radioURL)
	if errorSend != nil {
		return
	}
	return
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
		"-ac", "2",
		"-f", "opus",
		"-ar", "48000",
		"-vbr", "on",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("FFmpeg stdout error:", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		log.Println("FFmpeg start error:", err)
		return
	}

	buf := make([]byte, 1920*2*2)
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			break
		}
		vc.OpusSend <- buf[:n]
	}

	cmd.Wait()
}
