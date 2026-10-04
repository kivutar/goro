package game

import (
	"testing"
	"time"

	worldstate "github.com/kivutar/goro/world"
)

func TestSpecialNPCVisualForActor(t *testing.T) {
	tests := []struct {
		name  string
		actor worldstate.Actor
		want  specialNPCVisual
	}{
		{
			name:  "guild flag gr2 is normal actor model",
			actor: worldstate.Actor{Job: 722},
			want:  specialNPCVisualNone,
		},
		{
			name:  "city flag a remains sprite",
			actor: worldstate.Actor{Job: 1912},
			want:  specialNPCVisualNone,
		},
		{
			name:  "city flag b remains sprite",
			actor: worldstate.Actor{Job: 1913},
			want:  specialNPCVisualNone,
		},
		{
			name:  "sprite flag remains sprite",
			actor: worldstate.Actor{Job: 973},
			want:  specialNPCVisualNone,
		},
		{
			name:  "clear npc torch",
			actor: worldstate.Actor{Job: actorJobClearNPC, Name: "Bobbing Torch#7"},
			want:  specialNPCVisualTorch,
		},
		{
			name:  "clear npc firewood",
			actor: worldstate.Actor{Job: actorJobClearNPC, Name: "Wet Firewood#moc2"},
			want:  specialNPCVisualTorch,
		},
		{
			name:  "flame monster remains sprite",
			actor: worldstate.Actor{Job: 1869, HasObjectType: true, ObjectType: actorObjectTypeMob},
			want:  specialNPCVisualNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := specialNPCVisualForActor(tt.actor); got != tt.want {
				t.Fatalf("specialNPCVisualForActor(%+v) = %d, want %d", tt.actor, got, tt.want)
			}
		})
	}
}

func TestPersistentSpecialNPCWorldEffectKeepsActorCellAnchor(t *testing.T) {
	now := time.Unix(100, 0)
	entry := sceneActorDrawEntry{
		actor:  worldstate.Actor{ID: 700, X: 30, Y: 40, Job: actorJobClearNPC},
		worldX: 30.5,
		worldY: 40.5,
	}

	effect := persistentWorldEffectForActorEntry(effectTorch, entry, now)

	if effect.actorID != 700 || effect.effectID != effectTorch {
		t.Fatalf("effect identity = %+v", effect)
	}
	if effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect cell = %d,%d; want actor cell 30,40", effect.x, effect.y)
	}
	if !effect.starts.Equal(now) || !effect.expires.Equal(now.Add(24*time.Hour)) || effect.duration != 24*time.Hour {
		t.Fatalf("effect lifetime = starts %s expires %s duration %s", effect.starts, effect.expires, effect.duration)
	}
}
