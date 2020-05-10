package loganalyzer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyze(t *testing.T) {
	assert := require.New(t)
	{
		line := "[22:22:34] [Server thread/INFO]: <TontonAo> <troll> hey :) <yo>"
		res := Analyze(line)
		assert.IsType(NewMessagePayload{}, res)
		pl := res.(NewMessagePayload)
		assert.Equal("TontonAo", pl.Username)
		assert.Equal("<troll> hey :) <yo>", pl.Message)
	}
	{
		line := "[23:04:30] [Server thread/INFO]: TontonAo left the game"
		res := Analyze(line)
		assert.IsType(LeftGamePayload{}, res)
		pl := res.(LeftGamePayload)
		assert.Equal("TontonAo", pl.Username)
	}
	{
		line := "[23:04:20] [Server thread/INFO]: TontonAo joined the game"
		res := Analyze(line)
		assert.IsType(JoinGamePayload{}, res)
		pl := res.(JoinGamePayload)
		assert.Equal("TontonAo", pl.Username)
	}
	{
		line := "[23:14:58] [Server thread/INFO]: TontonAo has made the advancement [Ol' Betsy]"
		res := Analyze(line)
		assert.IsType(AdvancementPayload{}, res)
		pl := res.(AdvancementPayload)
		assert.Equal("TontonAo", pl.Username)
		assert.Equal("Ol' Betsy", pl.Advancement)
	}
	{
		line := "[23:19:42] [Server thread/INFO]: * TontonAo says stuff"
		res := Analyze(line)
		assert.IsType(MePayload{}, res)
		pl := res.(MePayload)
		assert.Equal("TontonAo", pl.Username)
		assert.Equal("says stuff", pl.Action)
	}

	{
		line := "[23:22:56] [Server thread/INFO]: TontonAo fell out of the world"
		res := Analyze(line)
		assert.IsType(DeathPayload{}, res)
		pl := res.(DeathPayload)
		assert.Equal("TontonAo", pl.Username)
		assert.Equal("fell out of the world", pl.Cause)
	}

	{
		line := "[00:02:29] [Server thread/INFO]: TontonAo was slain by Zombie using [Justice4all]"
		res := Analyze(line)
		assert.IsType(DeathPayload{}, res)
		pl := res.(DeathPayload)
		assert.Equal("TontonAo", pl.Username)
		assert.Equal("was slain by Zombie using [Justice4all]", pl.Cause)
	}
}
