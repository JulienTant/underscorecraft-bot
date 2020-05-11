package admin

import (
	"log"
	"time"
)

func (m *module) confirm(question string, yes func(), no func(), timeout func()) {
	dmsg, _ := m.discord.Send(m.adminChannel, question)
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
					yes()
					return
				case "❌":
					no()
					return
				}
			}
		}
		time.Sleep(time.Second)
	}
	timeout()
}

func (m *module) confirmGeneric(question string, yes func()) {
	m.confirm(question, yes, m.confirmGenericNo(), m.confirmGenericTimeout())
}

func (m *module) confirmGenericTimeout() func() {
	return func() {
		m.discord.Send(m.adminChannel, "Gave up")
	}
}

func (m *module) confirmGenericNo() func() {
	return func() {
		m.discord.Send(m.adminChannel, "Ok, i wont")
	}
}
