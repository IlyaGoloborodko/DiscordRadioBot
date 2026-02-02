package stream

import (
	"encoding/binary"
	"io"
	"log"
	"os/exec"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

func StartStreaming(vc *discordgo.VoiceConnection, url string) error {
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
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	enc, err := gopus.NewEncoder(48000, 2, gopus.Audio)
	if err != nil {
		return err
	}

	// Buffer for 20ms PCM frames
	pcmBuf := make([]int16, 960*2)

	for {
		select {
		case <-stopStreamChan:
			// получили сигнал остановки — завершаем цикл
			err := vc.Speaking(false)
			if err != nil {
				return err
			}
			err = cmd.Process.Kill() // убиваем ffmpeg
			if err != nil {
				return err
			}
			return nil
		default:
			// читаем PCM и отправляем в Discord
			if err := binary.Read(stdout, binary.LittleEndian, pcmBuf); err != nil {
				if err != io.EOF {
					log.Println("PCM read error:", err)
				}
				err := cmd.Wait()
				if err != nil {
					return err
				}
				return nil
			}
			opusFrame, err := enc.Encode(pcmBuf, len(pcmBuf)/2, len(pcmBuf)/2)
			if err != nil {
				continue
			}
			vc.OpusSend <- opusFrame
		}
	}
}
