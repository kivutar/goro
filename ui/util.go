package ui

import (
	"image"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

const (
	CursorActionDefault = 0
	CursorActionTalk    = 1
	CursorActionClick   = 2
	CursorActionRotate  = 4
	CursorActionAttack  = 5
	CursorActionWarp    = 7
	CursorActionPick    = 9
	CursorActionTarget  = 10
	CursorActionTarget2 = 11
	CursorActionNoWalk  = 13
)

const (
	actorObjectTypeMob       = 5
	actorObjectTypeNPC       = 6
	actorObjectTypeNPCABR    = 13
	actorObjectTypeNPCBionic = 14
)

type Context = client.Context

type AssetRenderer interface {
	DrawInventoryItemIcon(screen *render.Image, manager *res.Manager, item session.InventoryItem, x, y int)
	DrawSkillIcon(screen *render.Image, manager *res.Manager, skill session.Skill, x, y, size int)
	SkillIconImage(manager *res.Manager, skill session.Skill, size int) image.Image
	ItemInfoIllustrationImage(manager *res.Manager, item session.InventoryItem, width, height int) image.Image
	DrawEquipmentPreview(screen *render.Image, ctx client.Context, x, y, width, height int)
	EquipmentPreviewImage(ctx client.Context, width, height int) image.Image
}

type GameActions interface {
	UseShortcutSkill(ctx client.Context, skill session.Skill) error
	AddTeleportEffect(ctx client.Context)
}

func PointInRect(px, py, x, y, w, h int) bool {
	return px >= x && py >= y && px < x+w && py < y+h
}

func pointInRect(px, py, x, y, w, h int) bool {
	return PointInRect(px, py, x, y, w, h)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func trimRunes(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes <= 0 {
		return ""
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func selectedCharacter(s *session.Session) session.Character {
	if s == nil {
		return session.Character{Name: "Player"}
	}
	if s.Selected.ID != 0 {
		return s.Selected
	}
	for _, character := range s.Characters {
		if character.ID == s.CharID {
			return character
		}
	}
	if len(s.Characters) > 0 {
		return s.Characters[0]
	}
	return session.Character{ID: s.CharID, Name: "Player", Job: 0}
}

func skillByID(s *session.Session, skillID uint16) (session.Skill, bool) {
	if s == nil || skillID == 0 {
		return session.Skill{}, false
	}
	for _, skill := range s.Skills.List {
		if skill.ID == skillID {
			return skill, true
		}
	}
	return session.Skill{}, false
}
