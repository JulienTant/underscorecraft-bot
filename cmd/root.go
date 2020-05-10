package cmd

import (
	"bufio"
	"context"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/loganalyzer"
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

const MinecraftContainerNotFoundErrCode = -10
const MinecraftContainerNotRunning = -20

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
	Run: func(cmd *cobra.Command, args []string) {
		startedAt := time.Now().UTC()
		ctx := context.Background()

		botToken, err := cmd.Flags().GetString("bot-token")
		if err != nil {
			cmd.PrintErrf("Unable to get bot token: %s\n", err)
			os.Exit(-1)
		}
		channelID, err := cmd.Flags().GetString("channel-id")
		if err != nil {
			cmd.PrintErrf("Unable to get channel ID: %s\n", err)
			os.Exit(-1)
		}

		dClient, err := discord.NewClient(botToken, channelID)
		if err != nil {
			cmd.PrintErrf("Unable to create discord bot: %s\n", err)
			os.Exit(-1)
		}

		containerLabel, err := cmd.Flags().GetString("container-label")
		if err != nil {
			cmd.PrintErrf("Unable to get container label: %s\n", err)
			os.Exit(-1)
		}
		f := filters.NewArgs()
		f.Add("label", containerLabel)
		cmd.Println("looking for minecraft container")
		client, _ := client.NewEnvClient()
		containers, _ := client.ContainerList(ctx, types.ContainerListOptions{
			All:     true,
			Filters: f,
		})

		if len(containers) != 1 {
			cmd.PrintErrln("Unable to find minecraft container.")
			os.Exit(MinecraftContainerNotFoundErrCode)
		}
		cmd.Printf("found %s\n", containers[0].ID)

		if containers[0].State != "running" {
			cmd.PrintErrln("container is not running.")
			os.Exit(MinecraftContainerNotRunning)
		}

		attachedContainer, err := client.ContainerAttach(ctx, containers[0].ID, types.ContainerAttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		})
		if err != nil {
			log.Fatal(err)
		}

		dClient.OnNewMessage(func(username, msg string) {
			_, _ = attachedContainer.Conn.Write([]byte{'\n'})
			m := fmt.Sprintf(`tellraw @a [{"text":"<"},{"text":"[d]","bold":true,"color":"dark_purple","hoverEvent":{"action":"show_text","value":["",{"text":"Message From Discord"}]}},{"text":"%s> %s"}]`, username, msg)
			_, _ = attachedContainer.Conn.Write([]byte(m))
			_, _ = attachedContainer.Conn.Write([]byte{'\n'})
		})

		r := bufio.NewReader(attachedContainer.Reader)

		timer := time.NewTimer(10 * time.Second)
		for {
			select {
			case <-timer.C:
				panic("err?!")
			default:
				_ = attachedContainer.Conn.SetDeadline(time.Now().Add(5 * time.Second))

				s, err := r.ReadString('\n')
				if len(s) == 0 || err != nil && err != io.EOF {
					ccontainer, _ := client.ContainerInspect(ctx, containers[0].ID)
					cstartedAt, _ := time.Parse(time.RFC3339, ccontainer.State.StartedAt)
					if ccontainer.State.Running == false || cstartedAt.After(startedAt) {
						log.Fatal("something is wierd, rebooting")
					}
				}
				timer.Reset(10 * time.Second)
				res := loganalyzer.Analyze(s)
				switch pl := res.(type) {
				case loganalyzer.NewMessagePayload:
					_, err := dClient.Sendf("%s » %s", pl.Username, pl.Message)
					if err != nil {
						log.Printf("[err] send NewMessagePayload: %s", err)
					}
				case loganalyzer.JoinGamePayload:
					_, err := dClient.Sendf(":arrow_right: %s joined", pl.Username)
					if err != nil {
						log.Printf("[err] send JoinGamePayload: %s", err)
					}
				case loganalyzer.LeftGamePayload:
					_, err := dClient.Sendf(":arrow_left: %s left", pl.Username)
					if err != nil {
						log.Printf("[err] send LeftGamePayload: %s", err)
					}
				case loganalyzer.AdvancementPayload:
					_, err := dClient.Sendf(":muscle: %s has made the advancement %s", pl.Username, pl.Advancement)
					if err != nil {
						log.Printf("[err] send AdvancementPayload: %s", err)
					}
				case loganalyzer.ChallengePayload:
					_, err := dClient.Sendf(":muscle: %s has completed the challenge %s", pl.Username, pl.Advancement)
					if err != nil {
						log.Printf("[err] send ChallengePayload: %s", err)
					}
				case loganalyzer.MePayload:
					_, err := dClient.Sendf(`*\* %s %s*`, pl.Username, pl.Action)
					if err != nil {
						log.Printf("[err] send MePayload: %s", err)
					}
				case loganalyzer.DeathPayload:
					m, err := dClient.Sendf(":skull: %s %s", pl.Username, pl.Cause)
					if err != nil {
						log.Printf("[err] send DeathPayload: %s", err)
					} else {
						err = dClient.React(m, `🇫`)
						if err != nil {
							log.Printf("[err] add reaction to DeathPayload: %s", err)
						}
					}
				}
			}
		}

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
