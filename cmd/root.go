package cmd

import (
	"context"
	"docker-minecraft-to-discord/admin"
	"docker-minecraft-to-discord/chat"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().String("container-label", "com.mc2discord.is_server", "minecraft container name")
	rootCmd.Flags().String("chat-channel-id", "", "discord chat channel id")
	rootCmd.Flags().String("admin-channel-id", "", "discord admin channel id")
	rootCmd.Flags().String("bot-token", "", "discord bot token")

	_ = rootCmd.MarkFlagRequired("admin-channel-id")
	_ = rootCmd.MarkFlagRequired("chat-channel-id")
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
	startedAt := time.Now().UTC()
	ctx := context.Background()

	botToken, err := cmd.Flags().GetString("bot-token")
	if err != nil {
		log.Fatalf("get bot-token: %s", err)
	}

	chatChannelID, err := cmd.Flags().GetString("chat-channel-id")
	if err != nil {
		log.Fatalf("get chat-channel-id: %s", err)
	}

	adminChannelID, err := cmd.Flags().GetString("admin-channel-id")
	if err != nil {
		log.Fatalf("get admin-channel-id: %s", err)
	}

	containerLabel, err := cmd.Flags().GetString("container-label")
	if err != nil {
		log.Fatalf("get container-label: %s", err)
	}

	discordClient, err := discord.NewClient(botToken)
	if err != nil {
		log.Fatalf("discord.NewClient: %s", err)
	}

	inactivityDurationStr := os.Getenv("INACTIVITY_DURATION")
	inactivityDuration, err := time.ParseDuration(inactivityDurationStr)
	if inactivityDuration == time.Duration(0) || err != nil {
		log.Println("Setting default duration to one hour")
		inactivityDuration = time.Hour
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("NewDockerClient: %s", err)
	}

	log.Println("looking for minecraft container")
	container, err := dockerClient.GetContainerWithLabel(ctx, containerLabel)
	if err != nil {
		log.Fatalf("get container", err)
	}
	log.Printf("found %s", container.ID)

	attached, err := dockerClient.Attach(ctx, container)
	if err != nil {
		log.Fatal(err)
	}

	chatModule := chat.New(discordClient, chatChannelID, attached)
	discordClient.OnNewMessage(chatChannelID, chatModule.OnNewDiscordMessage)
	attached.OnNewMessage("discord <> mc chat", chatModule.OnNewAttachedMessage)

	adminModule := admin.New(discordClient, adminChannelID, dockerClient, container)
	discordClient.OnNewMessage(adminChannelID, adminModule.OnNewDiscordMessage)

	log.Fatalf("listen stopped: %s", attached.Listen(ctx, inactivityDuration, func() bool {
		return dockerClient.IsContainerAlive(ctx, container, startedAt)
	}))
}
