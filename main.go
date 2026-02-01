package main

import (
	"io"
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

	// Обработчик команд
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
		s.ChannelMessageSend(m.ChannelID, "Ошибка: не нашёл гильдию")
		return
	}

	// Находим голосовой канал пользователя
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

	// Стримим радио через ffmpeg
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
	// FFmpeg: берем поток и конвертируем в opus
	cmd := exec.Command("ffmpeg", "-i", url,
		"-f", "opus", "-ar", "48000", "-ac", "2", "pipe:1")

	stdout, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		return
	}

	opusBuf := make([]byte, 3840) // buffer frame
	for {
		n, err := stdout.Read(opusBuf)
		if err == io.EOF || err != nil {
			break
		}
		vc.OpusSend <- opusBuf[:n]
	}

	cmd.Wait()
}
