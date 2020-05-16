package maps

import (
	"docker-minecraft-to-discord/discord"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const MaxMarkersPerPlayer = 2

type module struct {
	discord      *discord.Client
	channelID    string
	jsonFilePath string
}

type action struct {
	prefix string
	help   string
	method func(user, s string)
}

var actions []action

func New(discord *discord.Client, channelID string, jsonFilePath string) *module {

	m := &module{
		discord:      discord,
		channelID:    channelID,
		jsonFilePath: jsonFilePath,
	}

	actions = []action{
		{
			prefix: "!help",
			help:   ": gives your the list of the commands",
			method: m.help,
		},
		{
			prefix: "!marker-add",
			help:   "<X Y Z> <name>: add a marker called <name> on given coordinates ",
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

func (m *module) help(_, _ string) {
	commands := []string{}
	for i := range actions {
		commands = append(commands, "**"+actions[i].prefix+"** "+actions[i].help)
	}
	m.discord.Send(m.channelID, strings.Join(commands, "\n"))
}

type Marker struct {
	ID          string `json:"id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Z           int    `json:"z"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *module) markerAdd(user, s string) {
	marker, err := markerFromString(user, s)
	if err != nil {
		m.discord.Send(m.channelID, "hughhh... I guess something is wrong with your coordinate?")
		return
	}

	file, err := ioutil.ReadFile(m.jsonFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			m.discord.Send(m.channelID, "unable to read the marker file :(")
			log.Printf("[err] unable to read marker file: %s", err)
			return
		}
		_, err = os.Create(m.jsonFilePath)
		if err != nil {
			log.Printf("[err] unable to create markers file: %s", err)
			return
		}
	}
	data := []Marker{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Print("[err] unable to unmarshal markers...meh")
	}
	nb := 0
	for i := range data {
		if data[i].Description == user {
			nb++
		}
		if nb == MaxMarkersPerPlayer {
			m.discord.Send(m.channelID, "You already have two markers.")
			return
		}
	}

	data = append(data, *marker)
	b, _ := json.Marshal(data)
	err = ioutil.WriteFile(m.jsonFilePath, b, 0644)
	if err != nil {
		log.Printf("[err] unable to save markers...")
	}
	m.discord.Send(m.channelID, "Marker added")
}

var markerRegex = regexp.MustCompile(`^(-?\d+)\s+(-?\d+)\s+(-?\d+)\s+(.*)$`)

func markerFromString(user, s string) (*Marker, error) {
	marker := Marker{
		ID:          "PlayerBase",
		Description: user,
	}
	markerRegexRes := markerRegex.FindStringSubmatch(strings.TrimSpace(s))
	if len(markerRegexRes) != 5 {
		return nil, errors.New("unable to parse marker")
	}

	marker.X, _ = strconv.Atoi(markerRegexRes[1])
	marker.Y, _ = strconv.Atoi(markerRegexRes[2])
	marker.Z, _ = strconv.Atoi(markerRegexRes[3])
	marker.Name = markerRegexRes[4]

	return &marker, nil
}

func (m *module) markerList(user, _ string) {
	file, err := ioutil.ReadFile(m.jsonFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			m.discord.Send(m.channelID, "unable to read the marker file :(")
			log.Printf("[err] unable to read marker file: %s", err)
			return
		}
		_, err = os.Create(m.jsonFilePath)
		if err != nil {
			log.Printf("[err] unable to create markers file: %s", err)
			return
		}
	}
	data := []Marker{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Print("[err] unable to unmarshal markers...meh")
	}

	hasMarkers := false
	sb := strings.Builder{}
	sb.WriteString("Here are your markers:\n")
	for i := range data {
		if data[i].Description == user {
			hasMarkers = true
			sb.WriteString(fmt.Sprintf("- **%s**: %d %d %d\n", data[i].Name, data[i].X, data[i].Y, data[i].Z))
		}
	}

	if !hasMarkers {
		m.discord.Send(m.channelID, "You have no markers")
		return
	}

	m.discord.Send(m.channelID, sb.String())
}

func (m *module) markerRemove(user, s string) {
	file, err := ioutil.ReadFile(m.jsonFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			m.discord.Send(m.channelID, "unable to read the marker file :(")
			log.Printf("[err] unable to read marker file: %s", err)
			return
		}
		_, err = os.Create(m.jsonFilePath)
		if err != nil {
			log.Printf("[err] unable to create markers file: %s", err)
			return
		}
	}
	data := []Marker{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Print("[err] unable to unmarshal markers...meh")
	}

	found := false
	i := 0
	for _, marker := range data {
		if !(marker.Description == user && marker.Name == s) {
			data[i] = marker
			i++
		} else {
			found = true
		}
	}
	data = data[:i]

	if !found {
		m.discord.Sendf(m.channelID, "You have no markers named %s", s)
		return
	}

	b, _ := json.Marshal(data)
	err = ioutil.WriteFile(m.jsonFilePath, b, 0644)
	if err != nil {
		log.Printf("[err] unable to save markers...")
	}

	m.discord.Send(m.channelID, "marker removed")
}

func (m *module) OnNewDiscordMessage(user string, msg string) {
	for i := range actions {
		if strings.HasPrefix(msg, actions[i].prefix) {
			actions[i].method(user, strings.TrimSpace(strings.TrimPrefix(msg, actions[i].prefix)))
		}
	}
}
