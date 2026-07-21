package app

import (
	"testing"

	"github.com/kivutar/goro/config"
)

func TestNewForceUserAIEnablesCompanionCustomAI(t *testing.T) {
	g, err := New(config.Config{
		DataDir: t.TempDir(),
		Window: config.WindowConfig{
			Width:  1280,
			Height: 720,
		},
		Packet: config.PacketConfig{
			ClientDate: 20080910,
		},
		Audio: config.AudioConfig{
			BGMVolume: 0.55,
			SFXVolume: 0.55,
		},
		Render: config.RenderConfig{
			GraphicsAPI: "vulkan",
			VSync:       true,
		},
		Gameplay: config.GameplayConfig{
			ForceUserAI: true,
		},
		Log: config.LogConfig{
			Level: "info",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !g.session.HomunculusCustomAI || !g.session.MercenaryCustomAI {
		t.Fatalf("custom AI flags = homunculus:%v mercenary:%v, want both true", g.session.HomunculusCustomAI, g.session.MercenaryCustomAI)
	}
}
