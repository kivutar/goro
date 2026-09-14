package game

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) applyNPCCutin(ctx client.Context, cutin network.NPCCutin) error {
	if ctx.Config.Headless {
		return nil
	}
	if cutin.Position == network.NPCCutinClear || cutin.Image == "" || cutin.Position > network.NPCCutinWindowless {
		m.ui.npcCutin.Apply(cutin, nil)
		return nil
	}
	texture, err := m.npcCutinTexture(ctx.Resources, cutin.Image)
	m.ui.npcCutin.Apply(cutin, texture)
	return err
}

func (m *WorldMode) npcCutinTexture(manager *res.Manager, resource string) (*render.Image, error) {
	resource = strings.TrimSpace(resource)
	if manager == nil || resource == "" {
		return nil, fmt.Errorf("cut-in resource manager unavailable")
	}
	candidates := res.NPCCutinTextureCandidates(resource)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("invalid cut-in resource %q", resource)
	}
	key := "__npc_cutin_" + strings.ToLower(strings.ReplaceAll(candidates[0], "\\", "/"))
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture := m.textures[key]; texture != nil {
		return texture, nil
	}
	if _, missed := m.textureMiss[key]; missed {
		return nil, nil
	}
	img, _, err := res.LoadImage(manager, candidates)
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil, err
	}
	texture := render.NewImageFromImage(img)
	m.textures[key] = texture
	return texture, nil
}

func (m *WorldMode) npcCutinPointerBlocked(ctx client.Context) bool {
	if ctx.Config.Render.NoUI || ctx.Input == nil {
		return false
	}
	width, height := ctx.ScreenSize()
	return m.ui.npcCutin.PointerBlocked(width, height, ctx.Input.MouseX, ctx.Input.MouseY)
}

func (m *WorldMode) bindNPCDialogLifecycle() {
	m.ui.npcDialog.SetCloseHandler(m.ui.npcCutin.Clear)
}
