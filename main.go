package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"layeh.com/gopus"
)

var allStreamURLs []string

var stopStreamChan = make(chan struct{})

func loadAllStreamURLs() error {
	stations, err := getAvailableRadios()
	if err != nil {
		return err
	}

	// Очищаем прошлые значения
	allStreamURLs = make([]string, 0, len(stations))

	// Наполняем только `url_resolved`
	for _, st := range stations {
		if st.StreamURL != "" {
			allStreamURLs = append(allStreamURLs, st.StreamURL)
		}
	}
	return nil
}

func init() {
	//godotenv.Load()
	//radioURL = os.Getenv("RADIO_URL")
	//if radioURL == "" {
	//	log.Fatal("RADIO_URL not set")
	//}
	godotenv.Load()
	if err := loadAllStreamURLs(); err != nil {
		log.Printf("failed to load station URLs: %v", err)
	} else {
		log.Printf("loaded %d stream URLs", len(allStreamURLs))
	}
}

type Mirror struct {
	URL string `json:"name"`
}

func getAvailableMirror() (string, error) {
	resp, err := http.Get("https://all.api.radio-browser.info/json/servers")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var mirrors []Mirror

	if err := json.NewDecoder(resp.Body).Decode(&mirrors); err != nil {
		return "", err
	}

	if len(mirrors) == 0 {
		return "", fmt.Errorf("no mirrors found")
	}

	return mirrors[0].URL, nil
}

type RadioStation struct {
	Name      string `json:"name"`
	StreamURL string `json:"url_resolved"`
	Homepage  string `json:"homepage,omitempty"`
	Country   string `json:"country,omitempty"`
	Tags      string `json:"tags,omitempty"`
	Bitrate   int    `json:"bitrate,omitempty"`
}

func getAvailableRadios() ([]RadioStation, error) {
	mirror, err := getAvailableMirror()
	if err != nil {
		return nil, fmt.Errorf("failed to get stations: %w", err)
	}
	resp, err := http.Get(fmt.Sprintf("https://%s/json/stations", mirror))
	//resp, err := http.Get(mirror)

	if err != nil {
		return nil, fmt.Errorf("failed to get stations: %w", err)
	}
	defer resp.Body.Close()

	var stations []RadioStation

	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, fmt.Errorf("failed to decode stations JSON: %w", err)
	}
	return stations, nil
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

	if strings.HasPrefix(content, "!stop") {
		stopRadio(s, m)
	}

	if strings.HasPrefix(content, "!disconnect") {
		disconnectChannel(s, m)
	}
}

func joinVoice(s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		//s.ChannelMessageSend(m.ChannelID, "Guild not found")
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
		//s.ChannelMessageSend(m.ChannelID, "Join a voice channel first")
		return
	}

	s.ChannelVoiceJoin(m.GuildID, vcID, false, true)
	//if err != nil {
	//	//s.ChannelMessageSend(m.ChannelID, "Failed to join voice channel")
	//	return
	//}

	//vc.Speaking(true)
	//s.ChannelMessageSend(m.ChannelID, "Joined voice!")
	log.Println("Connected to voice channel:", vcID)
}

func playRadio(s *discordgo.Session, m *discordgo.MessageCreate) {
	vc, found := findVoiceConnection(s, m.GuildID)
	if !found {
		//s.ChannelMessageSend(m.ChannelID, "First use !join")
		return
	}
	rand.Seed(time.Now().UnixNano())
	if len(allStreamURLs) == 0 {
		fmt.Println("Нет доступных потоков")
		return
	}
	idx := rand.Intn(len(allStreamURLs))
	radioURL := allStreamURLs[idx]

	select {
	case stopStreamChan <- struct{}{}:
	default:
	}

	go streamRadio(vc, radioURL)
	vc.Speaking(true)
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
		select {
		case <-stopStreamChan:
			// получили сигнал остановки — завершаем цикл
			vc.Speaking(false)
			cmd.Process.Kill() // убиваем ffmpeg
			return
		default:
			// читаем PCM и отправляем в Discord
			if err := binary.Read(stdout, binary.LittleEndian, pcmBuf); err != nil {
				if err != io.EOF {
					log.Println("PCM read error:", err)
				}
				cmd.Wait()
				return
			}
			opusFrame, err := enc.Encode(pcmBuf, len(pcmBuf)/2, len(pcmBuf)/2)
			if err != nil {
				log.Println("Opus encode error:", err)
				continue
			}
			vc.OpusSend <- opusFrame
		}
	}

	cmd.Wait()
	log.Println("Radio stream ended")
}

func stopRadio(s *discordgo.Session, m *discordgo.MessageCreate) {
	vc, found := findVoiceConnection(s, m.GuildID)
	if !found {
		//s.ChannelMessageSend(m.ChannelID, "Я не в голосовом канале.")
		return
	}

	// Останавливаем передачу аудио
	// Оповещаем Discord, что бот больше не говорит
	stopStreamChan <- struct{}{}
	vc.Speaking(false)

}

func disconnectChannel(s *discordgo.Session, m *discordgo.MessageCreate) {
	vc, found := findVoiceConnection(s, m.GuildID)
	if !found {
		//s.ChannelMessageSend(m.ChannelID, "Я не в голосовом канале.")
		return
	}
	// Закрываем голосовое соединение
	err := vc.Disconnect()
	if err != nil {
		//s.ChannelMessageSend(m.ChannelID, "Ошибка при отключении: "+err.Error())
		return
	}
}
