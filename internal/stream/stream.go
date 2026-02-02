package stream

import (
	"encoding/binary"
	"io"
	"log"
	"os/exec"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

func StreamRadio(vc *discordgo.VoiceConnection, url string) {
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
