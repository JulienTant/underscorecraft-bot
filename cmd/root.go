package cmd

import (
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/maps"
	"fmt"
	mcrcon "github.com/Kelwing/mc-rcon"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func init() {
	rootCmd.Flags().String("maps-channel-id", "", "discord maps channel id")
	rootCmd.Flags().String("bot-token", "", "discord bot token")
	rootCmd.Flags().String("server-address", "", "server address")
	rootCmd.Flags().String("server-rcon-pass", "", "server rcon password")

	_ = rootCmd.MarkFlagRequired("server-address")
	_ = rootCmd.MarkFlagRequired("server-rcon-pass")
	_ = rootCmd.MarkFlagRequired("maps-channel-id")
	_ = rootCmd.MarkFlagRequired("bot-token")
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

	mapsChannelID, err := cmd.Flags().GetString("maps-channel-id")
	if err != nil {
		log.Fatalf("get maps-channel-id: %s", err)
	}

	serverAddress, err := cmd.Flags().GetString("server-address")
	if err != nil {
		log.Fatalf("get maps-channel-id: %s", err)
	}

	serverRconPassword, err := cmd.Flags().GetString("server-rcon-pass")
	if err != nil {
		log.Fatalf("get maps-channel-id: %s", err)
	}

	discordClient, err := discord.NewClient(botToken)
	if err != nil {
		log.Fatalf("discord.NewClient: %s", err)
	}

	conn := new(mcrcon.MCConn)
	err = conn.Open(serverAddress, serverRconPassword)
	if err != nil {
		log.Fatalln("Open failed", err)
	}
	defer conn.Close()

	err = conn.Authenticate()
	if err != nil {
		log.Fatalln("Auth failed", err)
	}

	mapsModule := maps.New(discordClient, mapsChannelID, conn)
	discordClient.OnNewMessage(mapsChannelID, mapsModule.OnNewDiscordMessage)

	c := make(chan struct{})
	<-c
}
