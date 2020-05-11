package cmd

import (
	"context"
	"docker-minecraft-to-discord/admin"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"docker-minecraft-to-discord/loganalyzer"
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

	adminModule := admin.New(discordClient, adminChannelID, dockerClient, container, attached)

	discordClient.OnNewMessage(chatChannelID, func(username, msg string) {
		attached.SendString(fmt.Sprintf(`tellraw @a [{"text":"<"},{"text":"[d]","bold":true,"color":"dark_purple","hoverEvent":{"action":"show_text","value":["",{"text":"Message From Discord"}]}},{"text":"%s> %s"}]`, username, msg))
		if err != nil {
			log.Printf("[ERR] send string: %s", err)
		}
	})

	discordClient.OnNewMessage(adminChannelID, adminModule.OnNewDiscordMessage)

	attached.OnNewMessage("discord <> mc chat", func(s string) error {
		res := loganalyzer.Analyze(s)
		switch pl := res.(type) {
		case loganalyzer.NewMessagePayload:
			_, err := discordClient.Sendf(chatChannelID, "%s » %s", pl.Username, pl.Message)
			if err != nil {
				log.Printf("[err] send NewMessagePayload: %s", err)
			}
		case loganalyzer.JoinGamePayload:
			_, err := discordClient.Sendf(chatChannelID, ":arrow_right: %s joined", pl.Username)
			if err != nil {
				log.Printf("[err] send JoinGamePayload: %s", err)
			}
		case loganalyzer.LeftGamePayload:
			_, err := discordClient.Sendf(chatChannelID, ":arrow_left: %s left", pl.Username)
			if err != nil {
				log.Printf("[err] send LeftGamePayload: %s", err)
			}
		case loganalyzer.AdvancementPayload:
			_, err := discordClient.Sendf(chatChannelID, ":muscle: %s has made the advancement %s", pl.Username, pl.Advancement)
			if err != nil {
				log.Printf("[err] send AdvancementPayload: %s", err)
			}
		case loganalyzer.ChallengePayload:
			_, err := discordClient.Sendf(chatChannelID, ":muscle: %s has completed the challenge %s", pl.Username, pl.Advancement)
			if err != nil {
				log.Printf("[err] send ChallengePayload: %s", err)
			}
		case loganalyzer.MePayload:
			_, err := discordClient.Sendf(chatChannelID, `*\* %s %s*`, pl.Username, pl.Action)
			if err != nil {
				log.Printf("[err] send MePayload: %s", err)
			}
		case loganalyzer.DeathPayload:
			m, err := discordClient.Sendf(chatChannelID, ":skull: %s %s", pl.Username, pl.Cause)
			if err != nil {
				log.Printf("[err] send DeathPayload: %s", err)
			} else {
				err = discordClient.React(chatChannelID, m, `🇫`)
				if err != nil {
					log.Printf("[err] add reaction to DeathPayload: %s", err)
				}
			}
		}
		return nil
	})

	log.Fatalf("listen stopped: %s", attached.Listen(ctx, inactivityDuration, func() bool {
		return dockerClient.IsContainerAlive(ctx, container, startedAt)
	}))
}
