package admin

import (
	"context"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
)

type module struct {
	discord      *discord.Client
	adminChannel string

	dockerClient *docker.Client
	container    *docker.Container
}

type action struct {
	prefix string
	help   string
	method func(s string)
}

var actions []action

func New(discord *discord.Client, adminChannelID string, dockerClient *docker.Client, container *docker.Container) *module {

	m := &module{
		discord:      discord,
		adminChannel: adminChannelID,
		dockerClient: dockerClient,
		container:    container,
	}

	actions = []action{
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

func (m *module) OnNewDiscordMessage(_, _ string, msg string) {
	for i := range actions {
		if strings.HasPrefix(msg, actions[i].prefix) {
			actions[i].method(strings.TrimSpace(strings.TrimPrefix(msg, actions[i].prefix)))
		}
	}
}

func (m *module) bulkRemoveFromChannel(channelID string) {
	ch, err := m.discord.Session().Channel(channelID)
	if err != nil {
		m.discord.Sendf(m.adminChannel, "did not find channel %s", channelID)
	}

	m.confirmGeneric(fmt.Sprintf("Clean channel %s (you have 15s to react)?", ch.Name), func() {
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
		m.discord.Send(m.adminChannel, "Done")
	})
}

func (m *module) help(_ string) {
	commands := []string{}
	for i := range actions {
		commands = append(commands, "**"+actions[i].prefix+"** "+actions[i].help)
	}
	m.discord.Send(m.adminChannel, strings.Join(commands, "\n"))
}

func (m *module) rebootServer(_ string) {
	m.confirmGeneric("Are you sure you want to do that %s (you have 15s to react)?", func() {
		m.actualRcon("stop")
	})
}

func (m *module) whitelistAdd(mcUsername string) {
	m.actualRcon("whitelist add " + mcUsername)
}

func (m *module) whitelistRemove(mcUsername string) {
	m.actualRcon("whitelist remove " + mcUsername)
}

func (m *module) whitelistList(_ string) {
	m.actualRcon("whitelist list")
}

func (m *module) rcon(command string) {
	m.confirmGeneric(fmt.Sprintf(":scream: wow you're using rcon :scream: are you sure you want to execute %s", command), func() {
		m.actualRcon(command)
	})
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

	var bs []byte
	for {
		_ = a.Conn.SetDeadline(time.Now().Add(5 * time.Second))

		b, err := a.Reader.ReadBytes('\n')
		if err != nil {
			sb := strings.Builder{}
			sb.Write(bs[8 : len(bs)-1])
			m.discord.Send(m.adminChannel, sb.String())
			a.Close()
			return
		}
		bs = append(bs, b...)
	}
}
