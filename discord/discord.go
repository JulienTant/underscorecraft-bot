package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type client struct {
	s         *discordgo.Session
	channelID string
}

func NewClient(token, channelID string) (*client, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("creating discordgo: %w", err)
	}
	dg.ShouldReconnectOnError = true

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return nil, fmt.Errorf("opening discordgo: %w", err)
	}

	return &client{
		s:         dg,
		channelID: channelID,
	}, nil
}

func (c *client) OnNewMessage(fn func(string, string)) {
	c.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.ChannelID != c.channelID {
			return
		}

		c, _ := m.ContentWithMoreMentionsReplaced(s)
		if c == "" {
			return
		}
		fn(m.Author.Username, c)

	})
}

func (c *client) Close() error {
	return c.s.Close()
}

func (c *client) Send(msg string) (*discordgo.Message, error) {
	return c.Sendf(msg)
}

func (c *client) Sendf(msg string, args ...interface{}) (*discordgo.Message, error) {
	log.Printf(msg+" to %s", append(args, c.channelID)...)

	dmsg, err := c.s.ChannelMessageSend(c.channelID, fmt.Sprintf(msg, args...))
	if err != nil {
		return dmsg, fmt.Errorf("send msg: %w", err)
	}

	return dmsg, nil
}

func (c *client) React(msg *discordgo.Message, reaction string) error {
	return c.s.MessageReactionAdd(c.channelID, msg.ID, reaction)
}
