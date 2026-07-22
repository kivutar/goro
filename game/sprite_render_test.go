package game

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestNormalizeDirectionIndex(t *testing.T) {
	cases := map[int]int{
		0:  0,
		7:  7,
		8:  0,
		-1: 7,
		-9: 7,
	}
	for input, want := range cases {
		if got := normalizeDirectionIndex(input); got != want {
			t.Fatalf("normalizeDirectionIndex(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestSpriteDrawFilterUsesLinearSampling(t *testing.T) {
	if got := spriteDrawFilter(); got != render.FilterLinear {
		t.Fatalf("sprite draw filter = %v, want linear", got)
	}
}

func TestSpriteCompositionFilterKeepsSourcePixelsCrisp(t *testing.T) {
	if got := spriteCompositionFilter(); got != render.FilterNearest {
		t.Fatalf("sprite composition filter = %v, want nearest", got)
	}
}

func TestActorSpriteWorldZLiftsSpritesAboveTerrain(t *testing.T) {
	if got := actorSpriteWorldZ(12.5); math.Abs(got-12.7) > 1e-9 {
		t.Fatalf("actor sprite world z = %.3f, want 12.700", got)
	}
}

func TestSpriteDirectionFromWorldDirShowsBackForNorth(t *testing.T) {
	cases := map[int]int{
		0:  4,
		1:  3,
		2:  2,
		3:  1,
		4:  0,
		5:  7,
		6:  6,
		7:  5,
		8:  4,
		-1: 5,
	}
	for input, want := range cases {
		if got := spriteDirectionFromWorldDir(input); got != want {
			t.Fatalf("spriteDirectionFromWorldDir(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestSpriteDirectionFromWorldDirAccountsForCameraYaw(t *testing.T) {
	cases := []struct {
		name      string
		direction int
		cameraYaw float64
		want      int
	}{
		{name: "outdoor north", direction: 4, cameraYaw: 0, want: 0},
		{name: "indoor north", direction: 4, cameraYaw: -45, want: 7},
		{name: "indoor south", direction: 0, cameraYaw: -45, want: 3},
		{name: "east camera north", direction: 4, cameraYaw: 90, want: 2},
		{name: "rounded yaw", direction: 4, cameraYaw: 44, want: 1},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := spriteDirectionFromWorldDirForCamera(tt.direction, tt.cameraYaw); got != tt.want {
				t.Fatalf("spriteDirectionFromWorldDirForCamera(%d, %.1f) = %d, want %d", tt.direction, tt.cameraYaw, got, tt.want)
			}
		})
	}
}

func TestResolveSpriteActionPrefersFamilyAndDirection(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 16)}
	act.Actions[11] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	index, _, ok := resolveSpriteAction(act, spriteActionWalk, 3)
	if !ok {
		t.Fatal("expected action")
	}
	if index != 11 {
		t.Fatalf("index = %d, want 11", index)
	}
}

func TestResolveSpriteActionFallsBackToFamilyBase(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 16)}
	act.Actions[8] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	index, _, ok := resolveSpriteAction(act, spriteActionWalk, 4)
	if !ok {
		t.Fatal("expected action")
	}
	if index != 8 {
		t.Fatalf("index = %d, want 8", index)
	}
}

func TestResolveSpriteActionUsesDirectionForEightActionSprites(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 8)}
	for i := range act.Actions {
		act.Actions[i] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	}
	index, _, ok := resolveSpriteAction(act, spriteActionWalk, 5)
	if !ok {
		t.Fatal("expected action")
	}
	if index != 5 {
		t.Fatalf("index = %d, want direction action 5", index)
	}
}

func TestResolveSpriteActionUsesDirectIndexForCompactNonPCAct(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 5)}
	act.Actions[spriteActionNonPCAttack] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	index, _, ok := resolveSpriteAction(act, spriteActionNonPCAttack, 4)
	if !ok {
		t.Fatal("expected action")
	}
	if index != spriteActionNonPCAttack {
		t.Fatalf("index = %d, want %d", index, spriteActionNonPCAttack)
	}
}

func TestPreferNonPCActUpgradeForLegacyMonsterAct(t *testing.T) {
	if !preferNonPCActUpgrade(1002, 8, 72) {
		t.Fatal("expected richer monster ACT to upgrade legacy directional ACT")
	}
	if preferNonPCActUpgrade(1002, 40, 72) {
		t.Fatal("did not expect upgrade when current ACT already has action families")
	}
	if preferNonPCActUpgrade(47, 8, 72) {
		t.Fatal("did not expect NPC ACT upgrade")
	}
	if preferNonPCActUpgrade(1002, 8, 16) {
		t.Fatal("did not expect small candidate ACT upgrade")
	}
}

func TestGR2ResourcesDoNotUseSpriteFallbacks(t *testing.T) {
	tests := []struct {
		resource string
		want     bool
	}{
		{resource: "Guildflag90_1.gr2", want: true},
		{resource: `data\sprite\npc\empelium90_0.gr2`, want: true},
		{resource: "unknown.gr2", want: true},
		{resource: "OBJ_FLAG_A", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			if got := isGR2Resource(tt.resource); got != tt.want {
				t.Fatalf("isGR2Resource(%q) = %v, want %v", tt.resource, got, tt.want)
			}
		})
	}
}

func TestNoSpriteNPCJobsDoNotLoadFallbackSprite(t *testing.T) {
	manager := &res.Manager{}
	for _, job := range []int{actorJobWarpPortal, actorJobHiddenNPC, actorJobClearNPC} {
		if view, status := loadNonPCSpriteView(manager, job, "nonpc"); view != nil {
			t.Fatalf("job %d loaded fallback sprite: %s", job, status)
		}
	}
}

func TestPoringDeathActionRealData(t *testing.T) {
	manager := realDataManager(t)
	view, status := loadNonPCSpriteView(manager, 1002, "poring")
	if view == nil {
		t.Fatalf("Poring sprite failed: %s", status)
	}
	if len(view.act.Actions) < 40 {
		t.Fatalf("Poring ACT was not upgraded: actions=%d status=%s", len(view.act.Actions), status)
	}
	index, action, ok := resolveSpriteAction(view.act, spriteActionNonPCDeath, 0)
	if !ok {
		t.Fatalf("Poring death action missing: %s", status)
	}
	t.Logf("Poring death action index=%d frames=%d delay=%.1f source=%s status=%s", index, len(action.Animations), action.DelayMS, view.actSource, status)
	if index != spriteActionNonPCDeath*8 {
		t.Fatalf("Poring death action index=%d, want %d", index, spriteActionNonPCDeath*8)
	}
	if len(action.Animations) < 2 {
		t.Fatalf("Poring death frames=%d, want animated death", len(action.Animations))
	}
}

func TestActFitsSPRRejectsMissingFrame(t *testing.T) {
	spr := &res.SPR{Frames: make([]res.SPRFrame, 2), RGBAIndex: 2}
	act := &res.ACT{Actions: []res.ACTAction{{
		Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{Index: 1}}}},
	}}}
	if !actFitsSPR(act, spr) {
		t.Fatal("expected ACT to fit SPR")
	}
	act.Actions[0].Animations[0].Layers[0].Index = 2
	if actFitsSPR(act, spr) {
		t.Fatal("expected ACT with missing frame to be rejected")
	}
}

func TestSpriteMotionIndexUsesActionDelay(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	started := time.Unix(0, 0)
	got := spriteMotionIndex(action, started, started.Add(250*time.Millisecond), true)
	if got != 2 {
		t.Fatalf("motion index = %d, want 2", got)
	}
}

func TestHumanoidIdleUsesFirstBodyMotion(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	got := bodyMotionForState(action, spriteState{actionFamily: spriteActionIdle}, time.Unix(0, 0), time.Unix(0, 0).Add(250*time.Millisecond))
	if got != 0 {
		t.Fatalf("idle body motion = %d, want 0", got)
	}
}

func TestNonPCIdleBodyMotionAnimates(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	got := bodyMotionForState(action, spriteState{actionFamily: spriteActionIdle, loopIdle: true}, time.Unix(0, 0), time.Unix(0, 0).Add(250*time.Millisecond))
	if got != 2 {
		t.Fatalf("nonpc idle body motion = %d, want 2", got)
	}
}

func TestHumanoidWalkBodyMotionAnimates(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	got := bodyMotionForState(action, spriteState{actionFamily: spriteActionWalk}, time.Unix(0, 0), time.Unix(0, 0).Add(250*time.Millisecond))
	if got != 2 {
		t.Fatalf("walk body motion = %d, want 2", got)
	}
}

func TestHumanoidWalkBodyMotionScalesWithMoveSpeed(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    150,
	}
	started := time.Unix(0, 0)
	state := spriteState{actionFamily: spriteActionWalk, moveSpeedMS: 400}
	if got := bodyMotionForState(action, state, started, started.Add(399*time.Millisecond)); got != 0 {
		t.Fatalf("walk body motion before slow step = %d, want 0", got)
	}
	if got := bodyMotionForState(action, state, started, started.Add(400*time.Millisecond)); got != 1 {
		t.Fatalf("walk body motion after slow step = %d, want 1", got)
	}
}

func TestWalkBodyMotionUsesDistancePhase(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    150,
	}
	started := time.Unix(0, 0)
	state := spriteState{actionFamily: spriteActionWalk, moveSpeedMS: 400, walkDistance: 1}
	if got := bodyMotionForState(action, state, started, started.Add(399*time.Millisecond)); got != 1 {
		t.Fatalf("walk body motion = %d, want distance-driven 1", got)
	}
}

func TestWalkingSpriteStateIgnoresReadyFightAnimation(t *testing.T) {
	state := spriteState{
		actionFamily: spriteActionWalk,
		moving:       true,
		loop:         true,
	}

	if applyActorAnimationToSpriteState(&state, actorAnimation{actionFamily: spriteActionPCReadyFight, loop: true}, true) {
		t.Fatal("ready fight should not override walking state")
	}
	if state.actionFamily != spriteActionWalk || !state.moving {
		t.Fatalf("state = %+v, want still walking", state)
	}

	if !applyActorAnimationToSpriteState(&state, actorAnimation{actionFamily: spriteActionPCDeath}, true) {
		t.Fatal("death should override walking state")
	}
	if state.actionFamily != spriteActionPCDeath || state.moving {
		t.Fatalf("death state = %+v", state)
	}

	nonPC := spriteState{actionFamily: spriteActionWalk, moving: true, loop: true}
	if !applyActorAnimationToSpriteState(&nonPC, actorAnimation{actionFamily: spriteActionNonPCDeath}, false) {
		t.Fatal("non-pc death should override walking state")
	}
	if nonPC.actionFamily != spriteActionNonPCDeath || nonPC.moving {
		t.Fatalf("non-pc death state = %+v", nonPC)
	}
}

func TestTransientBodyMotionPlaysOnceFromStateStart(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	started := time.Unix(10, 0)
	got := bodyMotionForState(action, spriteState{actionFamily: spriteActionPCAttack2, started: started}, time.Unix(0, 0), started.Add(250*time.Millisecond))
	if got != 2 {
		t.Fatalf("attack body motion = %d, want 2", got)
	}
	got = bodyMotionForState(action, spriteState{actionFamily: spriteActionPCAttack2, started: started}, time.Unix(0, 0), started.Add(500*time.Millisecond))
	if got != 2 {
		t.Fatalf("finished attack body motion = %d, want final frame 2", got)
	}
}

func TestAttachmentDeltaMatchesAttachPointAttribute(t *testing.T) {
	base := res.ACTAnimation{Pos: []res.ACTPosition{
		{X: 1, Y: 2, Attr: 7},
		{X: 30, Y: 40, Attr: 2},
	}}
	attached := res.ACTAnimation{Pos: []res.ACTPosition{
		{X: 10, Y: 15, Attr: 2},
	}}
	x, y := attachmentDelta(base, attached)
	if x != 20 || y != 25 {
		t.Fatalf("attachment delta = (%d,%d), want (20,25)", x, y)
	}
}

func TestHumanoidIdleHeadMotionDoesNotCycle(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	if got := selectHeadMotion(spriteState{actionFamily: spriteActionIdle}, 2, action); got != 0 {
		t.Fatalf("idle head motion = %d, want 0", got)
	}
	if got := selectHeadMotion(spriteState{actionFamily: spriteActionIdle, headDir: 2, headTurn: true}, 0, action); got != 2 {
		t.Fatalf("idle turned head motion = %d, want 2", got)
	}
}

func TestHumanoidWalkHeadMotionFollowsBodyMotion(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	if got := selectHeadMotion(spriteState{actionFamily: spriteActionWalk}, 2, action); got != 2 {
		t.Fatalf("walk head motion = %d, want 2", got)
	}
}

func TestHumanoidTransientHeadMotionFollowsBodyMotion(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	if got := selectHeadMotion(spriteState{actionFamily: spriteActionPCAttack2}, 2, action); got != 2 {
		t.Fatalf("attack head motion = %d, want 2", got)
	}
}

func TestPlayerRenderLayerOrderMatchesDefaultOpenMidgardFallback(t *testing.T) {
	got := playerRenderLayerOrder(nil, 0, 0)
	want := [8]int{7, 0, 1, 4, 3, 2, 5, 6}
	if got != want {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestPlayerRenderLayerOrderKeepsBodyBeforeHead(t *testing.T) {
	imf := &res.IMF{Layers: make([]res.IMFLayer, 2)}
	for layer := range imf.Layers {
		imf.Layers[layer].Actions = []res.IMFAction{{
			Motions: []res.IMFMotion{{Priority: int32(1 - layer)}},
		}}
	}
	got := playerRenderLayerOrder(imf, 0, 0)
	bodyIndex, headIndex := -1, -1
	for index, layer := range got {
		if layer == 0 && bodyIndex < 0 {
			bodyIndex = index
		}
		if layer == 1 && headIndex < 0 {
			headIndex = index
		}
	}
	if bodyIndex < 0 || headIndex < 0 || bodyIndex > headIndex {
		t.Fatalf("body/head order = %v, body=%d head=%d", got, bodyIndex, headIndex)
	}
}

func TestPlayerRenderLayerOrderRestoresMissingHeadLayer(t *testing.T) {
	order := ensureRenderLayerPresent([8]int{0, 0, 5, 6, 7, 4, 3, 2}, 1)
	if order != [8]int{0, 1, 5, 6, 7, 4, 3, 2} {
		t.Fatalf("order = %v", order)
	}
}

func TestSpriteLayerCenterTreatsPositiveYAsScreenDown(t *testing.T) {
	_, y := spriteLayerCenter(5, 5, res.ACTLayer{Y: 3})
	if y != 8 {
		t.Fatalf("layer center Y = %.1f, want 8.0", y)
	}
}

func TestDebugPlayerSpriteBillboard(t *testing.T) {
	if os.Getenv("GORO_DEBUG_PLAYER_SPRITE") != "1" {
		t.Skip("set GORO_DEBUG_PLAYER_SPRITE=1")
	}
	manager := realDataManager(t)
	for _, sex := range []byte{0, 1} {
		view, status := loadHumanoidSpriteView(manager, 0, 1, sex, 0, 0, "debug player")
		if view == nil {
			t.Logf("sex=%d load failed: %s", sex, status)
			continue
		}
		billboard, ok := humanoidBillboardForState(view, spriteState{actionFamily: spriteActionIdle, direction: 0}, time.Now())
		if !ok {
			t.Logf("sex=%d billboard failed: %s", sex, status)
			continue
		}
		bodyIndex, bodyAction, ok := resolveSpriteAction(view.body.act, spriteActionIdle, 0)
		if !ok {
			t.Fatalf("sex=%d missing body action", sex)
		}
		minX, minY, maxX, maxY := spriteAnimationBounds(view.body, bodyAction.Animations[0], humanoidBillboardAnchorX, humanoidBillboardAnchorY, 0, 0)
		t.Logf("sex=%d %s action=%d bounds=(%.1f,%.1f)-(%.1f,%.1f) anchor=(%.0f,%.0f) image=%dx%d", sex, status, bodyIndex, minX, minY, maxX, maxY, billboard.anchorX, billboard.anchorY, humanoidBillboardWidth, humanoidBillboardHeight)
	}
}

func TestPlayerSpriteCompositionIncludesHeadWithWeapon(t *testing.T) {
	manager := realDataManager(t)
	view, status := loadHumanoidSpriteViewWithAppearance(manager, humanoidAppearance{
		job:    0,
		head:   1,
		sex:    0,
		weapon: 1201,
	}, "debug player")
	if view == nil {
		t.Skipf("debug player unavailable: %s", status)
	}
	bodyActionIndex, bodyAction, ok := resolveSpriteAction(view.body.act, spriteActionIdle, 0)
	if !ok {
		t.Fatalf("body action missing: %s", status)
	}
	order := playerRenderLayerOrder(view.imf, bodyActionIndex, 0)
	if !layerOrderContains(order, 1) {
		t.Fatalf("render order omits head layer: order=%v status=%s", order, status)
	}
	bodyAnim := bodyAction.Animations[0]
	headAnim, ok := actionAnimation(view.head.act, bodyActionIndex, 0)
	if !ok {
		t.Fatalf("head action missing: %s", status)
	}
	resolvedLayer := view.imf.LayerForPriority(1, bodyActionIndex, 0)
	if resolvedLayer < 0 {
		resolvedLayer = 1
	}
	if resolvedLayer < 0 || resolvedLayer >= len(headAnim.Layers) {
		t.Fatalf("head resolved layer %d out of %d: %s", resolvedLayer, len(headAnim.Layers), status)
	}
	pointX, pointY := view.imf.Point(resolvedLayer, bodyActionIndex, 0)
	dx, dy := attachmentDelta(bodyAnim, headAnim)
	drawLayer := visibleACTLayerIndex(headAnim, resolvedLayer, 1)
	if drawLayer < 0 {
		t.Fatalf("no visible head layer: resolved=%d status=%s", resolvedLayer, status)
	}
	if drawLayer != resolvedLayer {
		pointX, pointY = view.imf.Point(drawLayer, bodyActionIndex, 0)
	}
	layer := headAnim.Layers[drawLayer]
	minX, minY, maxX, maxY := spriteLayerBounds(view.head, layer, humanoidBillboardAnchorX+float64(pointX+dx), humanoidBillboardAnchorY+float64(pointY+dy))
	if minY > humanoidBillboardAnchorY-35 {
		t.Fatalf("head bounds too low: bounds=(%.1f,%.1f)-(%.1f,%.1f) point=(%d,%d) attach=(%d,%d) layer=%+v status=%s", minX, minY, maxX, maxY, pointX, pointY, dx, dy, layer, status)
	}
	t.Logf("head bounds=(%.1f,%.1f)-(%.1f,%.1f) point=(%d,%d) attach=(%d,%d) layer=%+v", minX, minY, maxX, maxY, pointX, pointY, dx, dy, layer)
}

func TestMercenaryHumanoidSpriteCompositionRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	cases := []struct {
		name   string
		job    int
		sex    byte
		death  int
		attack int
	}{
		{name: "archer", job: 6017, sex: 1, death: spriteActionNonPCDeath, attack: spriteActionPCAttack1},
		{name: "lancer", job: 6027, sex: 0, death: spriteActionPCDeath, attack: spriteActionPCAttack1},
		{name: "sword", job: 6037, sex: 0, death: spriteActionPCDeath, attack: spriteActionPCAttack1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view, status := loadMercenaryHumanoidSpriteView(manager, humanoidAppearance{job: tc.job, head: 1, sex: tc.sex}, "mercenary")
			if view == nil {
				t.Skipf("mercenary sprite unavailable: %s", status)
			}
			if view.body == nil || view.head == nil {
				t.Fatalf("mercenary view body=%t head=%t status=%s", view.body != nil, view.head != nil, status)
			}
			for _, action := range []int{tc.attack, spriteActionPCHurt, tc.death} {
				if _, _, ok := resolveSpriteAction(view.body.act, action, 0); !ok {
					t.Fatalf("missing body action=%d status=%s", action, status)
				}
			}
			if _, ok := humanoidBillboardForState(view, spriteState{actionFamily: tc.attack, direction: 0}, time.Now()); !ok {
				t.Fatalf("attack billboard unavailable status=%s", status)
			}
			if _, ok := humanoidBillboardForState(view, spriteState{actionFamily: tc.death, direction: 0}, time.Now()); !ok {
				t.Fatalf("death billboard unavailable action=%d status=%s", tc.death, status)
			}
		})
	}
}

func TestComposeSingleSpriteBillboardUsesAnimationBounds(t *testing.T) {
	view := &spriteView{
		spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  20,
				Height: 60,
				Data:   solidRGBAFrame(20, 60),
			}},
		},
		act:        &res.ACT{},
		images:     make(map[spriteFrameKey]*render.Image),
		billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}
	anim := res.ACTAnimation{Layers: []res.ACTLayer{{
		Index:   0,
		SPRType: res.SPRFrameRGBA,
		X:       3,
		Y:       -20,
		ScaleX:  1,
		ScaleY:  1,
		Color:   [4]float32{1, 1, 1, 1},
	}}}

	billboard, ok := composeSingleSpriteBillboard(view, anim)
	if !ok {
		t.Fatal("expected single sprite billboard")
	}
	if billboard.image.Bounds().Dx() != 28 || billboard.image.Bounds().Dy() != 68 {
		t.Fatalf("billboard size = %dx%d, want 28x68", billboard.image.Bounds().Dx(), billboard.image.Bounds().Dy())
	}
	if billboard.anchorX != 11 || billboard.anchorY != 54 {
		t.Fatalf("billboard anchor = %.1f, %.1f, want 11, 54", billboard.anchorX, billboard.anchorY)
	}
}

func TestCursorFrameBillboardUsesCompositionAnchorAsHotspot(t *testing.T) {
	view := &spriteView{
		spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  18,
				Height: 24,
				Data:   solidRGBAFrame(18, 24),
			}},
		},
		act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				X:       9,
				Y:       12,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		images:     make(map[spriteFrameKey]*render.Image),
		billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}

	billboard, ok := cursorFrameBillboard(view, 0, 0, 20, 24)
	if !ok {
		t.Fatal("expected cursor billboard")
	}
	if got, want := billboard.image.Bounds().Dx(), 26; got != want {
		t.Fatalf("cursor width = %d, want %d", got, want)
	}
	if got, want := billboard.image.Bounds().Dy(), 32; got != want {
		t.Fatalf("cursor height = %d, want %d", got, want)
	}
	if billboard.anchorX != 4 || billboard.anchorY != 4 {
		t.Fatalf("cursor anchor = %.1f, %.1f, want 4, 4", billboard.anchorX, billboard.anchorY)
	}
}

func TestCursorFrameBillboardDoesNotClipTallTargetCursor(t *testing.T) {
	view := &spriteView{
		spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  32,
				Height: 64,
				Data:   solidRGBAFrame(32, 64),
			}},
		},
		act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				X:       16,
				Y:       32,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		images:     make(map[spriteFrameKey]*render.Image),
		billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}

	billboard, ok := cursorFrameBillboard(view, 0, 0, 20, 50)
	if !ok {
		t.Fatal("expected cursor billboard")
	}
	if got, want := billboard.image.Bounds().Dy(), 72; got != want {
		t.Fatalf("cursor height = %d, want %d", got, want)
	}
	if got, want := billboard.anchorY, 4.0; got != want {
		t.Fatalf("cursor anchorY = %.1f, want %.1f", got, want)
	}
}

func solidRGBAFrame(width, height int) []byte {
	data := make([]byte, width*height*4)
	for i := 0; i < len(data); i += 4 {
		data[i+0] = 255
		data[i+3] = 255
	}
	return data
}

func layerOrderContains(order [8]int, layer int) bool {
	for _, value := range order {
		if value == layer {
			return true
		}
	}
	return false
}

func spriteAnimationBounds(view *spriteView, anim res.ACTAnimation, anchorX, anchorY float64, posX, posY int32) (float64, float64, float64, float64) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, layer := range anim.Layers {
		if layer.Index < 0 {
			continue
		}
		frameIndex := int(layer.Index)
		if layer.SPRType == res.SPRFrameRGBA {
			frameIndex += view.spr.RGBAIndex
		}
		if frameIndex < 0 || frameIndex >= len(view.spr.Frames) {
			continue
		}
		frame := view.spr.Frames[frameIndex]
		cx := anchorX + float64(posX) + float64(layer.X)
		cy := anchorY + float64(posY) + float64(layer.Y)
		w := float64(frame.Width)
		h := float64(frame.Height)
		minX = math.Min(minX, cx-w/2)
		minY = math.Min(minY, cy-h/2)
		maxX = math.Max(maxX, cx+w/2)
		maxY = math.Max(maxY, cy+h/2)
	}
	return minX, minY, maxX, maxY
}

func spriteLayerBounds(view *spriteView, layer res.ACTLayer, centerX, centerY float64) (float64, float64, float64, float64) {
	frameIndex := int(layer.Index)
	if layer.SPRType == res.SPRFrameRGBA {
		frameIndex += view.spr.RGBAIndex
	}
	if frameIndex < 0 || frameIndex >= len(view.spr.Frames) {
		return math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	}
	width := float64(view.spr.Frames[frameIndex].Width)
	height := float64(view.spr.Frames[frameIndex].Height)
	scaleX := math.Abs(float64(layer.ScaleX))
	scaleY := math.Abs(float64(layer.ScaleY))
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}
	layerCenterX, layerCenterY := spriteLayerCenter(centerX, centerY, layer)
	halfW := width * scaleX / 2
	halfH := height * scaleY / 2
	return layerCenterX - halfW, layerCenterY - halfH, layerCenterX + halfW, layerCenterY + halfH
}
