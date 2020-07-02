package maps

import (
	"context"
	"docker-minecraft-to-discord/discord"
	"docker-minecraft-to-discord/docker"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
)

const MaxMarkersPerPlayer = 5

type module struct {
	discord      *discord.Client
	channelID    string
	dockerClient *docker.Client
	container    *docker.Container
}

type action struct {
	prefix string
	help   string
	method func(userID, user, s string)
}

var actions []action

func New(discord *discord.Client, channelID string, dockerClient *docker.Client, container *docker.Container) *module {

	m := &module{
		discord:      discord,
		channelID:    channelID,
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
	return fmt.Sprintf(`dmarker add "%s" id:%s world:%s x:%f y:64 z:%f set:Bases`, m.Name, m.ID, m.World, m.X, m.Z)
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
		sb.WriteString(fmt.Sprintf("- **%s**: %.1f %.1f (%s)\n", markers[i].Name, markers[i].X, markers[i].Z, markers[i].World))
	}

	m.discord.Send(m.channelID, sb.String())
}

func (m *module) markerRemove(userID, _, s string) {
	/*file, err := ioutil.ReadFile(m.jsonFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			m.discord.Send(m.channelID, "Unable to read the marker file :( Please reach TontonAo")
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

	m.discord.Send(m.channelID, "Marker removed. It'll be dropped from the map in maximum a minute...")*/
}

func (m *module) OnNewDiscordMessage(userid, user, msg string) {
	for i := range actions {
		if strings.HasPrefix(msg, actions[i].prefix) {
			actions[i].method(userid, user, strings.TrimSpace(strings.TrimPrefix(msg, actions[i].prefix)))
		}
	}
}

func (m *module) actualRcon(command string) string {
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
			a.Close()
			return sb.String()
		}
		bs = append(bs, b...)
	}
}
