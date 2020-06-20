package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type Client struct {
	s *discordgo.Session
}

func NewClient(token string) (*Client, error) {
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

	return &Client{
		s: dg,
	}, nil
}

func (c *Client) Session() *discordgo.Session {
	return c.s
}

func (c *Client) OnNewMessage(channelID string, fn func(string, string)) {
	c.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if channelID != "all" && m.ChannelID != channelID {
			return
		}

		c, _ := m.ContentWithMoreMentionsReplaced(s)
		if c == "" {
			return
		}

		fn(m.Member.Nick, c)

	})
}

func (c *Client) Close() error {
	return c.s.Close()
}

func (c *Client) Send(chatChannelID string, msg string) (*discordgo.Message, error) {
	return c.Sendf(chatChannelID, msg)
}

func (c *Client) Sendf(chatChannelID string, msg string, args ...interface{}) (*discordgo.Message, error) {
	log.Printf(msg+" to %s", append(args, chatChannelID)...)

	dmsg, err := c.s.ChannelMessageSend(chatChannelID, fmt.Sprintf(msg, args...))
	if err != nil {
		return dmsg, fmt.Errorf("send msg: %w", err)
	}

	return dmsg, nil
}

func (c *Client) React(chatChannelID string, msg *discordgo.Message, reaction string) error {
	return c.s.MessageReactionAdd(chatChannelID, msg.ID, reaction)
}
