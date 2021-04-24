package maps

import (
	"docker-minecraft-to-discord/discord"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	mcrcon "github.com/Kelwing/mc-rcon"
)

const MaxMarkersPerPlayer = 5

type module struct {
	discord   *discord.Client
	channelID string
	rcon      *mcrcon.MCConn
}

type action struct {
	prefix string
	help   string
	method func(userID, user, s string)
}

var actions []action

func New(discord *discord.Client, channelID string, rcon *mcrcon.MCConn) *module {

	m := &module{
		discord:   discord,
		channelID: channelID,
		rcon:      rcon,
	}

	actions = []action{
		{
			prefix: "!help",
			help:   ": gives your the list of the commands",
			method: m.help,
		},
		{
			prefix: "!marker-add",
			help:   "<overworld|nether|end> <X Z> <name>: add a marker called <name> on given coordinates",
			method: m.markerAdd,
		},
		{
			prefix: "!markers",
			help:   ": list your markers",
			method: m.markerList,
		},
		{
			prefix: "!marker-remove",
			help:   "<name>: remove the marker.",
			method: m.markerRemove,
		},
	}

	return m
}

type Marker struct {
	ID    string
	Name  string
	World string
	X     float64
	Z     float64
}

func (m Marker) ToAddCommand() string {
	return fmt.Sprintf(`dmarker add "%s" id:%s world:%s x:%f y:64 z:%f set:Bases markup:true`, m.Name, m.ID, m.World, m.X, m.Z)
}

func (m Marker) ToRemoveCommand() string {
	return fmt.Sprintf(`dmarker delete id:%s set:Bases`, m.ID)
}

func (m *module) help(_, _, _ string) {
	commands := []string{}
	for i := range actions {
		commands = append(commands, "**"+actions[i].prefix+"** "+actions[i].help)
	}
	m.discord.Send(m.channelID, strings.Join(commands, "\n"))
}

func (m *module) markerAdd(userID, _, s string) {
	marker, err := markerFromString(userID, s)
	if err != nil {
		m.discord.Send(m.channelID, "hughhh... I guess something is wrong with your coordinate?")
		return
	}

	mks := m.getMarkers(userID)
	if len(mks) >= MaxMarkersPerPlayer {
		m.discord.Sendf(m.channelID, "You already have %d markers", len(mks))
		return
	}

	res := m.actualRcon(marker.ToAddCommand())
	if !strings.Contains(res, "Added marker") {
		log.Println("unable to add marker: ", res)
		m.discord.Send(m.channelID, "hughhh... i can't add the marker :/ Reach an admin!")
		return
	}

	m.discord.Send(m.channelID, "Marker added!")
	return
}

var markerRegex = regexp.MustCompile(`^(overworld|nether|end)\s+(-?\d+)\s+(-?\d+)\s+(.*)$`)

func markerFromString(userID, s string) (*Marker, error) {
	marker := Marker{
		ID: userID + "_" + fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	markerRegexRes := markerRegex.FindStringSubmatch(strings.TrimSpace(s))
	if len(markerRegexRes) != 5 {
		return nil, errors.New("unable to parse marker")
	}

	switch markerRegexRes[1] {
	case "overworld":
		marker.World = "world"
	case "nether":
		marker.World = "world_nether"
	case "end":
		marker.World = "world_the_end"
	}

	marker.X, _ = strconv.ParseFloat(markerRegexRes[2], 64)
	marker.Z, _ = strconv.ParseFloat(markerRegexRes[3], 64)
	marker.Name = markerRegexRes[4]

	return &marker, nil
}

func (m *module) getMarkers(userID string) []Marker {
	markersAsString := strings.Split(m.actualRcon("dmarker list set:Bases"), "\n")
	log.Println(markersAsString)

	var markers []Marker
	for _, v := range markersAsString {
		if !strings.HasPrefix(v, userID) {
			continue
		}

		m := Marker{ID: v[:strings.Index(v, ":")]}
		parts := strings.Split(v[strings.Index(v, ":")+2:], ", ")
		for _, p := range parts {
			kv := strings.Split(p, ":")
			switch kv[0] {
			case "label":
				m.Name = strings.Trim(kv[1], `"`)
			case "world":
				switch kv[1] {
				case "world":
					m.World = "overworld"
				case "world_nether":
					m.World = "nether"
				case "world_the_end":
					m.World = "the_end"
				}
			case "x":
				x, _ := strconv.ParseFloat(kv[1], 64)
				m.X = x
			case "z":
				z, _ := strconv.ParseFloat(kv[1], 64)
				m.Z = z
			}
		}
		markers = append(markers, m)
	}
	return markers
}

func (m *module) markerList(userID, _, _ string) {
	markers := m.getMarkers(userID)

	if len(markers) == 0 {
		m.discord.Send(m.channelID, "You have no markers!")
		return
	}

	sb := strings.Builder{}
	sb.WriteString("Here are your markers:\n")
	for i := range markers {
		sb.WriteString(fmt.Sprintf("- **%s**: x=%.0f z=%.0f (%s)\n", markers[i].Name, markers[i].X, markers[i].Z, markers[i].World))
	}

	m.discord.Send(m.channelID, sb.String())
}

func (m *module) markerRemove(userID, _, s string) {
	markers := m.getMarkers(userID)

	for i := range markers {
		if markers[i].Name == s {
			res := m.actualRcon(markers[i].ToRemoveCommand())
			if !strings.Contains(res, "Deleted marker") {
				log.Println(res)
				m.discord.Send(m.channelID, "Unable to remove marker. Contact an admin")
				return
			}
			m.discord.Send(m.channelID, "Marker removed.")
			return
		}
	}

	m.discord.Sendf(m.channelID, "You have no markers named %s", s)
}

func (m *module) OnNewDiscordMessage(userid, user, msg string) {
	for i := range actions {
		if strings.HasPrefix(msg, actions[i].prefix) {
			actions[i].method(userid, user, strings.TrimSpace(strings.TrimPrefix(msg, actions[i].prefix)))
		}
	}
}

func (m *module) actualRcon(command string) string {
	log.Println("send command", command)
	res, err := m.rcon.SendCommand(command)
	if err != nil {
		log.Println("[err] SendCommand", err)
	}
	log.Println("got result", res)

	return res
}
