package db

import "time"

const EffectFrameDuration = time.Second / 60

// EffectHitRing holds the Prim3DCylinder motion in cells per original-client tick.
type EffectHitRing struct {
	Speed, HeightSpeed, HeightAcceleration float64
	InitialDistance                        float64
	FadeOutFrame                           int
	RandomImpactOffset                     bool
}

// EffectHitParticles uses the original random speed range, in hundredths of
// a native unit per tick. One cell is five native units.
type EffectHitParticles struct {
	SpeedMin, SpeedRange int
}

func hitRingComponent(texture string, frames int, inner, outer, height, speed float64, fadeOutFrame int) EffectComponent {
	return EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "HitRing", TextureName: texture,
		Duration: time.Duration(frames) * EffectFrameDuration, FrameDelay: EffectFrameDuration,
		AlphaMax: 254.0 / 255, BottomSize: inner / 5, TopSize: outer / 5,
		Height: height / 5, PosZ: 2, AngleX: 90, CircleSides: 10, AttachedEntity: true,
		HitRing: &EffectHitRing{Speed: speed / 5, FadeOutFrame: fadeOutFrame},
	}
}

// CRagEffect::Hit3/Hit4: growing cylinders followed by three-segment SPR trails.
func legacyHitEffectSpec(arrow bool) EffectSpec {
	inner, count, speedMin, speedRange := 1.5, 8, 80, 120
	heightAccel, sound := 0.2, "effect\\ef_hit3.wav"
	if arrow {
		inner, count, speedMin, speedRange = 0.5, 5, 60, 90
		heightAccel, sound = 0.15, "effect\\ef_hit4.wav"
	}
	ring := hitRingComponent("lens2", 15, inner, 4, 0, 0.7, 10)
	ring.HitRing.HeightSpeed = 0.25 / 5
	ring.HitRing.HeightAcceleration = heightAccel / 5
	components := []EffectComponent{}
	if !arrow {
		core := hitRingComponent("lens2", 15, 1.5, 1.5, 0, 0.7, 10)
		core.HitRing.HeightSpeed = 0.5 / 5
		core.HitRing.HeightAcceleration = 0.2 / 5
		components = append(components, core)
	}
	components = append(components, ring, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "HitParticles", SpriteFile: "particle1",
		Duration: 30 * EffectFrameDuration, FrameDelay: EffectFrameDuration,
		Duplicate: count, PosZ: 2, AttachedEntity: true,
		HitParticles: &EffectHitParticles{SpeedMin: speedMin, SpeedRange: speedRange},
	})
	return EffectSpec{Duration: 30 * EffectFrameDuration, SFX: []string{sound}, Components: components}
}
