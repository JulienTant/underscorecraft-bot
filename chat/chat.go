package chat

import (
	"bytes"
	"context"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"docker-minecraft-to-discord/loganalyzer"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	mcpinger "github.com/Raqbit/mc-pinger"
)

type module struct {
	discord *discord.Client
	channel string

	attached *docker.Attached
}

func New(discord *discord.Client, channel string, attached *docker.Attached) *module {
	return &module{
		discord: discord,
		channel: channel,

		attached: attached,
	}
}

func (m *module) OnNewDiscordMessage(username, msg string) {
	b, _ := json.Marshal(username)
	secureUsername := string(bytes.Trim(b, `"`))
	b, _ = json.Marshal(msg)
	secureMsg := string(bytes.Trim(b, `"`))

	err := m.attached.SendString(fmt.Sprintf(`tellraw @a [{"text":"<"},{"text":"[d]","bold":true,"color":"dark_purple","hoverEvent":{"action":"show_text","value":["",{"text":"Message From Discord"}]}},{"text":"%s> %s"}]`, secureUsername, secureMsg))
	if err != nil {
		log.Printf("[ERR] send string: %s", err)
	}
}

func (m *module) OnNewAttachedMessage(s string) error {
	res := loganalyzer.Analyze(s)
	switch pl := res.(type) {
	case loganalyzer.NewMessagePayload:
		_, err := m.discord.Sendf(m.channel, "%s » %s", pl.Username, pl.Message)
		if err != nil {
			log.Printf("[err] send NewMessagePayload: %s", err)
		}
	case loganalyzer.JoinGamePayload:
		_, err := m.discord.Sendf(m.channel, ":arrow_right: %s joined", pl.Username)
		if err != nil {
			log.Printf("[err] send JoinGamePayload: %s", err)
		}
	case loganalyzer.LeftGamePayload:
		_, err := m.discord.Sendf(m.channel, ":arrow_left: %s left", pl.Username)
		if err != nil {
			log.Printf("[err] send LeftGamePayload: %s", err)
		}
	case loganalyzer.AdvancementPayload:
		_, err := m.discord.Sendf(m.channel, ":muscle: %s has made the advancement %s", pl.Username, pl.Advancement)
		if err != nil {
			log.Printf("[err] send AdvancementPayload: %s", err)
		}
	case loganalyzer.ChallengePayload:
		_, err := m.discord.Sendf(m.channel, ":muscle: %s has completed the challenge %s", pl.Username, pl.Advancement)
		if err != nil {
			log.Printf("[err] send ChallengePayload: %s", err)
		}
	case loganalyzer.MePayload:
		_, err := m.discord.Sendf(m.channel, `*\* %s %s*`, pl.Username, pl.Action)
		if err != nil {
			log.Printf("[err] send MePayload: %s", err)
		}
	case loganalyzer.DeathPayload:
		msg, err := m.discord.Sendf(m.channel, ":skull: %s %s", pl.Username, pl.Cause)
		if err != nil {
			log.Printf("[err] send DeathPayload: %s", err)
		} else {
			err = m.discord.React(m.channel, msg, `🇫`)
			if err != nil {
				log.Printf("[err] add reaction to DeathPayload: %s", err)
			}
		}
	}
	return nil
}

func (m *module) RefreshOnlinePlayers(ctx context.Context) {
	pinger := mcpinger.New("underscorecraft.com", 25565)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			info, err := pinger.Ping()
			if err != nil {
				log.Println(err)
			} else {
				ch, err := m.discord.Session().Channel(m.channel)
				if err != nil {
					continue
				}
				m.discord.Session().ChannelEditComplex(m.channel, &discordgo.ChannelEdit{
					Topic:    fmt.Sprintf("%d players online - IP: underscorecraft.com", info.Players.Online),
					Position: ch.Position,
				})
			}
			time.Sleep(15 * time.Second)
		}
	}
}
