package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestApplyLocalGuildDetailsInfersMasterFromSelectedCharacter(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Name: "Arcer"},
		Guild:    session.Guild{IsMaster: false},
	}

	applyLocalGuildDetails(client.Context{Session: s}, network.GuildInfo{
		GuildID:    1,
		GuildName:  "Mandala",
		MasterName: "Arcer",
	})

	if !s.Guild.IsMaster {
		t.Fatal("selected guild master should get master access from guild info")
	}
}

func TestApplyLocalGuildDetailsClearsMasterWhenSelectedCharacterIsNotMaster(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Name: "Kivutar"},
		Guild:    session.Guild{IsMaster: true},
	}

	applyLocalGuildDetails(client.Context{Session: s}, network.GuildInfo{
		GuildID:    1,
		GuildName:  "Mandala",
		MasterName: "Arcer",
	})

	if s.Guild.IsMaster {
		t.Fatal("non-master selected character should lose master access from guild info")
	}
}
