package cmd

import (
	"bufio"
	"context"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/loganalyzer"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().String("container-label", "com.mc2discord.is_server", "minecraft container name")
	rootCmd.Flags().String("channel-id", "", "discord channel id")
	rootCmd.Flags().String("bot-token", "", "discord bot token")

	_ = rootCmd.MarkFlagRequired("channel-id")
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

	channelID, err := cmd.Flags().GetString("channel-id")
	if err != nil {
		log.Fatalf("get channel-id: %s", err)
	}

	containerLabel, err := cmd.Flags().GetString("container-label")
	if err != nil {
		log.Fatalf("get container-label: %s", err)
	}

	discordClient, err := discord.NewClient(botToken, channelID)
	if err != nil {
		log.Fatalf("discord.NewClient: %s", err)
	}

	inactivityDurationStr := os.Getenv("INACTIVITY_DURATION")
	inactivityDuration, err := time.ParseDuration(inactivityDurationStr)
	if inactivityDuration == time.Duration(0) || err != nil {
		log.Println("Setting default duration to one hour")
		inactivityDuration = time.Hour
	}

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("NewDockerClient: %s", err)
	}

	log.Println("looking for minecraft container")
	container, err := getContainer(ctx, dockerClient, containerLabel)
	if err != nil {
		log.Fatalf("getContainer: %s", err)
	}
	log.Printf("found %s", container.ID)

	attachedContainer, err := dockerClient.ContainerAttach(ctx, container.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	discordClient.OnNewMessage(func(username, msg string) {
		_, _ = attachedContainer.Conn.Write([]byte{'\n'})
		m := fmt.Sprintf(`tellraw @a [{"text":"<"},{"text":"[d]","bold":true,"color":"dark_purple","hoverEvent":{"action":"show_text","value":["",{"text":"Message From Discord"}]}},{"text":"%s> %s"}]`, username, msg)
		_, _ = attachedContainer.Conn.Write([]byte(m))
		_, _ = attachedContainer.Conn.Write([]byte{'\n'})
	})

	r := bufio.NewReader(attachedContainer.Reader)
	lastThing := time.Now()
	for {
		_ = attachedContainer.Conn.SetDeadline(time.Now().Add(5 * time.Second))

		if lastThing.Add(inactivityDuration).After(time.Now()) {
			log.Fatalf("No activity for %s, restart", inactivityDuration)
		}

		s, err := r.ReadString('\n')
		if len(s) == 0 || err != nil && err != io.EOF {
			ccontainer, _ := dockerClient.ContainerInspect(ctx, container.ID)
			cstartedAt, _ := time.Parse(time.RFC3339, ccontainer.State.StartedAt)
			if ccontainer.State.Running == false || cstartedAt.After(startedAt) {
				log.Fatal("No message & the server has been restarted, i'll restart too...")
			}
			continue
		}
		res := loganalyzer.Analyze(s)
		switch pl := res.(type) {
		case loganalyzer.NewMessagePayload:
			_, err := discordClient.Sendf("%s » %s", pl.Username, pl.Message)
			if err != nil {
				log.Printf("[err] send NewMessagePayload: %s", err)
			}
		case loganalyzer.JoinGamePayload:
			_, err := discordClient.Sendf(":arrow_right: %s joined", pl.Username)
			if err != nil {
				log.Printf("[err] send JoinGamePayload: %s", err)
			}
		case loganalyzer.LeftGamePayload:
			_, err := discordClient.Sendf(":arrow_left: %s left", pl.Username)
			if err != nil {
				log.Printf("[err] send LeftGamePayload: %s", err)
			}
		case loganalyzer.AdvancementPayload:
			_, err := discordClient.Sendf(":muscle: %s has made the advancement %s", pl.Username, pl.Advancement)
			if err != nil {
				log.Printf("[err] send AdvancementPayload: %s", err)
			}
		case loganalyzer.ChallengePayload:
			_, err := discordClient.Sendf(":muscle: %s has completed the challenge %s", pl.Username, pl.Advancement)
			if err != nil {
				log.Printf("[err] send ChallengePayload: %s", err)
			}
		case loganalyzer.MePayload:
			_, err := discordClient.Sendf(`*\* %s %s*`, pl.Username, pl.Action)
			if err != nil {
				log.Printf("[err] send MePayload: %s", err)
			}
		case loganalyzer.DeathPayload:
			m, err := discordClient.Sendf(":skull: %s %s", pl.Username, pl.Cause)
			if err != nil {
				log.Printf("[err] send DeathPayload: %s", err)
			} else {
				err = discordClient.React(m, `🇫`)
				if err != nil {
					log.Printf("[err] add reaction to DeathPayload: %s", err)
				}
			}
		}
		lastThing = time.Now()
	}

}

func getContainer(ctx context.Context, docker *client.Client, containerLabel string) (*types.Container, error) {
	f := filters.NewArgs()
	f.Add("label", containerLabel)
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: f,
	})

	if err != nil {
		return nil, fmt.Errorf("containerList: %w", err)
	}

	if len(containers) != 1 {
		return nil, errors.New("unable to find minecraft container")
	}

	if containers[0].State != "running" {
		return nil, errors.New("container is not running")
	}

	return &containers[0], nil
}
