package chat

import (
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"docker-minecraft-to-discord/loganalyzer"
	"encoding/json"
	"fmt"
	"log"
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
	secureUsername := string(b)
	b, _ = json.Marshal(msg)
	secureMsg := string(b)

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
