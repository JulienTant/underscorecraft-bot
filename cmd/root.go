package cmd

import (
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/maps"
	"docker-minecraft-to-discord/pterodactyl"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/james4k/rcon"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().String("maps-channel-id", "", "discord maps channel id")
	rootCmd.Flags().String("bot-token", "", "discord bot token")
	rootCmd.Flags().String("server-address", "", "server address")
	rootCmd.Flags().String("server-rcon-pass", "", "server rcon password")
	rootCmd.Flags().String("pterodactyl-address", "", "pterodactyl address")
	rootCmd.Flags().String("pterodactyl-api-key", "", "pterodactyl api key")
	rootCmd.Flags().String("pterodactyl-server-id", "", "pterodactyl server id")

	_ = rootCmd.MarkFlagRequired("server-address")
	_ = rootCmd.MarkFlagRequired("server-rcon-pass")
	_ = rootCmd.MarkFlagRequired("maps-channel-id")
	_ = rootCmd.MarkFlagRequired("bot-token")
	_ = rootCmd.MarkFlagRequired("pterodactyl-address")
	_ = rootCmd.MarkFlagRequired("pterodactyl-api-key")
	_ = rootCmd.MarkFlagRequired("pterodactyl-server-id")
}

var rootCmd = &cobra.Command{
	Version: "0.0.1",
	Use:     "docker-minecraft-to-discord",
	Short:   "docker-minecraft-to-discord is a minecraft <-> discord gateway",
	Long:    `A discord <-> minecraft gateway that allows your non playing members to communicate with playing ones!`,
	Run:     runRootCmd,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runRootCmd(cmd *cobra.Command, _ []string) {
	botToken, err := cmd.Flags().GetString("bot-token")
	if err != nil {
		log.Fatalf("get bot-token: %s", err)
	}

	discordClient, err := discord.NewClient(botToken)
	if err != nil {
		log.Fatalf("discord.NewClient: %s", err)
	}

	serverAddress, err := cmd.Flags().GetString("server-address")
	if err != nil {
		log.Fatalf("get server-address: %s", err)
	}

	serverRconPassword, err := cmd.Flags().GetString("server-rcon-pass")
	if err != nil {
		log.Fatalf("get server-rcon-pass: %s", err)
	}
	conn, err := rcon.Dial(serverAddress, serverRconPassword)
	if err != nil {
		log.Fatalln("dial failed", err)
	}
	defer conn.Close()

	pteroAddress, err := cmd.Flags().GetString("pterodactyl-address")
	if err != nil {
		log.Fatalf("get pterodactyl-address: %s", err)
	}

	pteroApiKey, err := cmd.Flags().GetString("pterodactyl-api-key")
	if err != nil {
		log.Fatalf("get pterodactyl-api-key: %s", err)
	}
	pterodactyl := pterodactyl.NewClient(pteroAddress, pteroApiKey)

	pteroServerID, err := cmd.Flags().GetString("pterodactyl-server-id")
	if err != nil {
		log.Fatalf("get pterodactyl-server-id: %s", err)
	}

	mapsChannelID, err := cmd.Flags().GetString("maps-channel-id")
	if err != nil {
		log.Fatalf("get maps-channel-id: %s", err)
	}

	mapsModule := maps.New(discordClient, mapsChannelID, conn, pterodactyl, pteroServerID)
	discordClient.OnNewMessage(mapsChannelID, mapsModule.OnNewDiscordMessage)

	log.Println("Bot started")
	var w sync.WaitGroup
	w.Add(1)
	var sig chan os.Signal
	sig = make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		w.Done()
	}()
	w.Wait()
}
