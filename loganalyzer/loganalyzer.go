package loganalyzer

import (
	"regexp"
	"strings"
)

var serverThreadLogRgx = regexp.MustCompile(`\[Server thread/INFO]: (.*)`)
var newMsgRgx = regexp.MustCompile(`^<([a-zA-Z0-9_]+)> (.*)$`)
var leftGameRgx = regexp.MustCompile(`^([a-zA-Z0-9_]+) left the game$`)
var joinGameRgx = regexp.MustCompile(`^([a-zA-Z0-9_]+) joined the game$`)
var advancementRgx = regexp.MustCompile(`^([a-zA-Z0-9_]+) has made the advancement \[(.*)]$`)
var challengeRgx = regexp.MustCompile(`^([a-zA-Z0-9_]+) has completed the challenge \[(.*)]$`)
var meRgx = regexp.MustCompile(`^\* ([a-zA-Z0-9_]+) (.*)$`)

type UnknownPayload struct {
}

type NewMessagePayload struct {
	Username, Message string
}

type LeftGamePayload struct {
	Username string
}

type JoinGamePayload struct {
	Username string
}

type AdvancementPayload struct {
	Username, Advancement string
}

type ChallengePayload struct {
	Username, Advancement string
}

type MePayload struct {
	Username, Action string
}

type DeathPayload struct {
	Username, Cause string
}

func Analyze(line string) interface{} {
	res := serverThreadLogRgx.FindStringSubmatch(line)
	if len(res) != 2 {
		return UnknownPayload{}
	}

	secondPart := strings.TrimSpace(res[1])
	msgRgxRes := newMsgRgx.FindStringSubmatch(secondPart)
	if len(msgRgxRes) == 3 {
		return NewMessagePayload{Username: msgRgxRes[1], Message: msgRgxRes[2]}
	}

	joinGameRes := joinGameRgx.FindStringSubmatch(secondPart)
	if len(joinGameRes) == 2 {
		return JoinGamePayload{Username: joinGameRes[1]}
	}

	leftGameRes := leftGameRgx.FindStringSubmatch(secondPart)
	if len(leftGameRes) == 2 {
		return LeftGamePayload{Username: leftGameRes[1]}
	}

	advancementGameRes := advancementRgx.FindStringSubmatch(secondPart)
	if len(advancementGameRes) == 3 {
		return AdvancementPayload{Username: advancementGameRes[1], Advancement: advancementGameRes[2]}
	}

	challengeGameRes := challengeRgx.FindStringSubmatch(secondPart)
	if len(challengeGameRes) == 3 {
		return ChallengePayload{Username: challengeGameRes[1], Advancement: challengeGameRes[2]}
	}

	meGameRes := meRgx.FindStringSubmatch(secondPart)
	if len(meGameRes) == 3 {
		return MePayload{Username: meGameRes[1], Action: meGameRes[2]}
	}

	for _, v := range deathMsgContains {
		deathGameRes := v.FindStringSubmatch(secondPart)
		if len(deathGameRes) >= 3 {
			return DeathPayload{Username: deathGameRes[1], Cause: deathGameRes[2]}
		}
	}

	return UnknownPayload{}
}
