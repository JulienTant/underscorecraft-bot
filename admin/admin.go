package admin

import (
	"context"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/docker/docker/api/types"
)

type module struct {
	discord      *discord.Client
	adminChannel string

	dockerClient *docker.Client
	container    *docker.Container
	attached     *docker.Attached
}

type method struct {
	prefix string
	help   string
	method func(s string)
}

var methods []method

func New(discord *discord.Client, adminChannelID string, dockerClient *docker.Client, container *docker.Container, attached *docker.Attached) *module {

	m := &module{
		discord:      discord,
		adminChannel: adminChannelID,
		dockerClient: dockerClient,
		container:    container,
		attached:     attached,
	}

	methods = []method{
		{
			prefix: "!help",
			help:   ": gives your the list of the commands",
			method: m.help,
		},
		{
			prefix: "!clear-discord-channel",
			help:   "<channel-id>: remove all messages from a channel",
			method: m.bulkRemoveFromChannel,
		},
		{
			prefix: "!reboot-server",
			help:   ": reboot the server...!!!!",
			method: m.rebootServer,
		},
		{
			prefix: "!whitelist-add",
			help:   "<username>: adds a user to the whitelist",
			method: m.whitelistAdd,
		},
		{
			prefix: "!whitelist-remove",
			help:   "<username>: removes a user to the whitelist",
			method: m.whitelistRemove,
		},
		{
			prefix: "!whitelist-list",
			help:   ": list whitelisted players",
			method: m.whitelistList,
		},
		{
			prefix: "!rcon",
			help:   ": start an rcon command. USE WITH CAUTION!",
			method: m.rcon,
		},
	}

	return m
}

func (m *module) OnNewDiscordMessage(_ string, msg string) {
	for i := range methods {
		if strings.HasPrefix(msg, methods[i].prefix) {
			methods[i].method(msg)
		}
	}
}

func (m *module) bulkRemoveFromChannel(msg string) {
	channelID := strings.TrimSpace(strings.TrimPrefix(msg, "!clear-discord-channel"))
	ch, err := m.discord.Session().Channel(channelID)
	if err != nil {
		m.discord.Sendf(m.adminChannel, "did not find channel %s", channelID)
	}
	dmsg, _ := m.discord.Sendf(m.adminChannel, "Clean channel %s (you have 15s to react)?", ch.Name)
	m.discord.React(m.adminChannel, dmsg, "✅")
	m.discord.React(m.adminChannel, dmsg, "❌")

	for cnt := 0; cnt < 15; cnt++ {
		log.Printf("iteration %d...", cnt+1)
		dmsg, err = m.discord.Session().ChannelMessage(m.adminChannel, dmsg.ID)
		if err != nil {
			log.Printf("[err] at read channel msg: %s", err)
			break
		}
		log.Printf("dmsg refreshed, has %d reactions...", len(dmsg.Reactions))
		for i := range dmsg.Reactions {
			log.Printf("reaction %s has cnt of %d...", dmsg.Reactions[i].Emoji.Name, dmsg.Reactions[i].Count)

			if dmsg.Reactions[i].Count >= 2 {
				log.Println("dealing with it")

				switch dmsg.Reactions[i].Emoji.Name {
				case "✅":
					stop := false
					for !stop {
						msgs, err := m.discord.Session().ChannelMessages(channelID, 100, "", "", "")
						if err != nil || len(msgs) == 0 {
							if err != nil {
								log.Printf("[err] at read channel msgs: %s", err)
							}
							stop = true
							break
						}
						msgsID := []string{}
						for j := range msgs {
							msgsID = append(msgsID, msgs[j].ID)
						}
						err = m.discord.Session().ChannelMessagesBulkDelete(channelID, msgsID)
						if err != nil {
							log.Printf("[err] at bulk delete: %s", err)
						}
					}
					_, err = m.discord.Send(m.adminChannel, "Done")
					return
				case "❌":
					_, err = m.discord.Send(m.adminChannel, "Ok, i wont")
					if err != nil {
						log.Printf("[err] channel delete message: %s", err)
					}
					return
				}
			}
		}
		time.Sleep(time.Second)
	}
	_, err = m.discord.Send(m.adminChannel, "Gave up")

}

func (m *module) help(s string) {
	commands := []string{}
	for i := range methods {
		commands = append(commands, "**"+methods[i].prefix+"** "+methods[i].help)
	}
	m.discord.Send(m.adminChannel, strings.Join(commands, "\n"))
}

func (m *module) rebootServer(s string) {
	dmsg, _ := m.discord.Send(m.adminChannel, "Are you sure you want to do that %s (you have 15s to react)?")
	m.discord.React(m.adminChannel, dmsg, "✅")
	m.discord.React(m.adminChannel, dmsg, "❌")

	for cnt := 0; cnt < 15; cnt++ {
		log.Printf("iteration %d...", cnt+1)
		dmsg, err := m.discord.Session().ChannelMessage(m.adminChannel, dmsg.ID)
		if err != nil {
			log.Printf("[err] at read channel msg: %s", err)
			break
		}
		log.Printf("dmsg refreshed, has %d reactions...", len(dmsg.Reactions))
		for i := range dmsg.Reactions {
			log.Printf("reaction %s has cnt of %d...", dmsg.Reactions[i].Emoji.Name, dmsg.Reactions[i].Count)

			if dmsg.Reactions[i].Count >= 2 {
				log.Println("dealing with it")

				switch dmsg.Reactions[i].Emoji.Name {
				case "✅":
					m.attached.SendString("stop")
					_, err = m.discord.Send(m.adminChannel, "Ok, stop command sent")
					return
				case "❌":
					_, err = m.discord.Send(m.adminChannel, "Ok, i wont")
					return
				}
			}
		}
		time.Sleep(time.Second)
	}
	m.discord.Send(m.adminChannel, "Gave up")
}

func (m *module) whitelistAdd(msg string) {
	mcUsername := strings.TrimSpace(strings.TrimPrefix(msg, "!whitelist-add"))
	m.attached.SendString("whitelist add " + mcUsername)
	m.discord.Send(m.adminChannel, "Added to whitelist")
}

func (m *module) whitelistRemove(msg string) {
	mcUsername := strings.TrimSpace(strings.TrimPrefix(msg, "!whitelist-remove"))
	m.attached.SendString("whitelist remove " + mcUsername)
	m.discord.Send(m.adminChannel, "Removed from whitelist")
}

func (m *module) whitelistList(_ string) {
	m.actualRcon("whitelist list")
}

func (m *module) rcon(msg string) {
	command := strings.TrimSpace(strings.TrimPrefix(msg, "!rcon"))

	dmsg, _ := m.discord.Sendf(m.adminChannel, ":scream: wow you're using rcon :scream: are you sure you want to execute %s", command)
	m.discord.React(m.adminChannel, dmsg, "✅")
	m.discord.React(m.adminChannel, dmsg, "❌")

	for cnt := 0; cnt < 15; cnt++ {
		dmsg, err := m.discord.Session().ChannelMessage(m.adminChannel, dmsg.ID)
		if err != nil {
			log.Printf("[err] at read channel msg: %s", err)
			break
		}

		for i := range dmsg.Reactions {
			if dmsg.Reactions[i].Count >= 2 {
				switch dmsg.Reactions[i].Emoji.Name {
				case "✅":
					m.actualRcon(command)
					return
				case "❌":
					_, err = m.discord.Send(m.adminChannel, "Ok, i wont")
					return
				}
			}
		}
		time.Sleep(time.Second)
	}
	m.discord.Send(m.adminChannel, "Gave up")

}

func (m *module) actualRcon(command string) {
	cmd := append([]string{"rcon-cli"}, strings.Split(command, " ")...)
	id, err := m.dockerClient.InnerClient().ContainerExecCreate(context.Background(), m.container.ID, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	})
	if err != nil {
		log.Println("[err] whitelist ContainerExecCreate", err)
	}

	a, err := m.dockerClient.InnerClient().ContainerExecAttach(context.Background(), id.ID, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	})
	if err != nil {
		log.Println("[err] whitelist ContainerExecAttach", err)
	}

	str := ""
	for {
		_ = a.Conn.SetDeadline(time.Now().Add(5 * time.Second))

		s, err := a.Reader.ReadString('\n')
		if err != nil {
			m.discord.Send(m.adminChannel, str)
			a.Close()
			return
		}

		str += strings.TrimFunc(s, func(r rune) bool {
			return !unicode.IsGraphic(r)
		})
	}
}
