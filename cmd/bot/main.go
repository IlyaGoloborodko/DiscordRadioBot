package main

import (
	"discordAudio/internal/discord"
	"discordAudio/internal/radio"
	"log"
	"os"
	"os/signal"
	"syscall"

	"discordAudio/internal/config"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func init() {
	//godotenv.Load()
	//radioURL = os.Getenv("RADIO_URL")
	//if radioURL == "" {
	//	log.Fatal("RADIO_URL not set")
	//}
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}
	if err := radio.LoadAllStations(); err != nil {
		log.Printf("failed to load station URLs: %v", err)
	} else {
		log.Printf("loaded %d stations", len(radio.AllStations))
	}
}

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}
	config.DebugGuildID = os.Getenv("DEBUG_GUIID")

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("error creating Discord session,", err)
	}

	dg.AddHandler(discord.MessageHandler)

	err = discord.RegisterCommands(dg)
	if err != nil {
		log.Fatal("error register Discord commands,", err)
	}

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
