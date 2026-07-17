package game

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

const petCaptureSkillID = 0xFFFF

const (
	petStateInit        uint8 = 0
	petStateFriendly    uint8 = 1
	petStateHunger      uint8 = 2
	petStateAccessory   uint8 = 3
	petStatePerformance uint8 = 4
)

const (
	petTalkFeeding = iota
	petTalkHunting
	petTalkDanger
	petTalkDead
	petTalkNormal
	petTalkPerformanceSpecial
	petTalkLevelUp
	petTalkPerformance1
	petTalkPerformance2
	petTalkPerformance3
	petTalkConnect
)

const (
	petTalkFriendlyThreshold = 900
	petTalkCooldown          = 10 * time.Second
)

var petEmotionTable = [5][5][7]int{
	{
		{32, 32, 29, 29, 7, 23, 28},
		{32, 32, 29, 29, 7, 7, 5},
		{20, 32, -1, 32, 5, 20, 10},
		{33, -1, -1, 10, 20, 10, 6},
		{18, -1, -1, 20, 33, 2, 6},
	},
	{
		{20, 32, 29, 29, 32, 32, 5},
		{20, 32, -1, 32, 29, 32, 32},
		{33, -1, -1, 10, 20, 10, 6},
		{18, -1, -1, 20, 33, 2, 6},
		{15, -1, -1, 19, 3, 2, 21},
	},
	{
		{32, 32, -1, 32, 29, 5, 32},
		{20, -1, -1, 10, 10, 10, 20},
		{33, -1, -1, 20, 33, 2, 6},
		{18, -1, -1, 19, 18, 2, 21},
		{15, 21, -1, 23, 4, 33, 21},
	},
	{
		{32, -1, -1, 10, 10, 10, 20},
		{32, -1, -1, 20, 20, 20, 10},
		{20, -1, -1, 19, 21, 2, 6},
		{33, 21, 19, 23, 3, 33, 21},
		{18, 21, 26, 28, 30, 22, 18},
	},
	{
		{23, -1, -1, 20, 20, 20, 9},
		{32, -1, -1, 19, 20, 0, 0},
		{32, 21, 19, 23, 18, 33, 21},
		{20, 21, 26, 28, 3, 18, 18},
		{20, 21, 26, 28, 30, 18, 30},
	},
}

type petCaptureState struct {
	active  bool
	started time.Time
}

func petCaptureTargetSkill() session.Skill {
	return session.Skill{
		ID:    petCaptureSkillID,
		Name:  "Capture Monster",
		Type:  skillTargetPet,
		Range: 9,
	}
}

func (m *WorldMode) startPetCapture(ctx client.Context) {
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	m.pendingPetCapture = petCaptureState{active: true, started: time.Now()}
	m.ui.console.AddSystemMessage("Select a monster to capture.")
	glog.Debugf("pet capture target pending")
}

func (m *WorldMode) cancelPetCapture(source string) {
	if !m.pendingPetCapture.active {
		return
	}
	m.pendingPetCapture = petCaptureState{}
	glog.Debugf("pet capture canceled source=%s", source)
}

func (m *WorldMode) cancelPetCaptureFromInput(ctx client.Context) bool {
	if !m.pendingPetCapture.active || ctx.Input == nil {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEscape) && !ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		return false
	}
	m.cancelPetCapture("input")
	return true
}

func (m *WorldMode) handlePetCaptureClick(ctx client.Context, projection sceneProjection, now time.Time) bool {
	if !m.pendingPetCapture.active || ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return false
	}
	skill := petCaptureTargetSkill()
	actor, ok := clickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		if x, y, groundOK := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); groundOK {
			glog.Debugf("pet capture canceled by ground click mouse=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, x, y)
			m.cancelPetCapture("ground-click")
			return true
		}
		glog.Debugf("pet capture target miss mouse=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY)
		return true
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Capture failed: not connected.")
		return true
	}
	m.pendingPetCapture = petCaptureState{}
	m.openPetSlotMachine(ctx, actor.ID)
	glog.Debugf("pet capture target selected target=%d name=%q job=%d object_type=%d", actor.ID, actor.Name, actor.Job, actor.ObjectType)
	return true
}

func (m *WorldMode) applyPetCaptureResult(ctx client.Context, result network.PetCaptureResult) {
	if m.petSlotMachine.active {
		m.petSlotMachine.setResult(result.Success)
	} else if result.Success {
		m.ui.console.AddBlueMessage("Pet capture succeeded.")
	} else {
		m.ui.console.AddErrorMessage("Pet capture failed.")
	}
	glog.Debugf("pet capture result success=%t", result.Success)
}

func (m *WorldMode) applyPetEggList(ctx client.Context, list network.PetEggList) {
	glog.Debugf("pet egg list indexes=%v", list.Indexes)
	if len(list.Indexes) == 0 {
		m.ui.console.AddErrorMessage("No pet eggs available.")
		return
	}
	m.ui.petEggWindow.OpenList(ctx, list)
}

func (m *WorldMode) applyPetProperty(ctx client.Context, property network.PetProperty) {
	glog.Debugf("pet property name=%q level=%d fullness=%d relationship=%d accessory=%d job=%d modified=%t", property.Name, property.Level, property.Fullness, property.Relationship, property.AccessoryID, property.Job, property.Modified)
	if !m.hasPetProperty {
		m.petOldFullness = property.Fullness
	} else {
		m.petOldFullness = m.petProperty.Fullness
	}
	m.petProperty = property
	m.hasPetProperty = true
	if m.petInfoRequested || m.ui.petInfoWindow.IsOpen() {
		m.petInfoRequested = false
		m.ui.petInfoWindow.OpenInfo(ctx, property)
	}
}

func (m *WorldMode) applyPetFeedResult(ctx client.Context, result network.PetFeedResult) {
	name := fmt.Sprintf("item %d", result.ItemID)
	if ctx.Resources != nil {
		if resolved, ok := ctx.Resources.ItemDisplayName(int(result.ItemID), true); ok && strings.TrimSpace(resolved) != "" {
			name = resolved
		}
	}
	if result.Success {
		m.ui.console.AddBlueMessage("Fed pet with %s.", name)
		m.sendPetFeedEmotion(ctx)
		m.sendPetTalk(ctx, petTalkFeeding)
	} else {
		m.ui.console.AddErrorMessage("Failed to feed pet with %s.", name)
	}
	glog.Debugf("pet feed result success=%t item=%d", result.Success, result.ItemID)
}

func (m *WorldMode) applyPetStateChange(ctx client.Context, change network.PetStateChange) {
	glog.Debugf("pet state type=%d id=%d data=%d", change.Type, change.ID, change.Data)
	switch change.Type {
	case petStateInit:
		m.petID = change.ID
	case petStateFriendly:
		m.petProperty.Relationship = uint16(change.Data)
	case petStateHunger:
		m.petOldFullness = m.petProperty.Fullness
		m.petProperty.Fullness = uint16(change.Data)
	case petStateAccessory:
		m.applyPetAccessoryChange(ctx, change.ID, change.Data)
	case petStatePerformance:
		m.startPetPerformance(ctx, change.ID, change.Data)
	}
	if m.hasPetProperty && m.ui.petInfoWindow.IsOpen() {
		m.ui.petInfoWindow.OpenInfo(ctx, m.petProperty)
	}
}

func (m *WorldMode) applyPetAccessoryChange(ctx client.Context, id uint32, accessoryID uint32) {
	if accessoryID == 0 {
		delete(m.petAccessoryIDs, id)
		return
	}
	if m.petAccessoryIDs == nil {
		m.petAccessoryIDs = make(map[uint32]uint32)
	}
	m.petAccessoryIDs[id] = accessoryID
	if ctx.World != nil {
		if actor, ok := ctx.World.Actors[id]; ok {
			_ = m.nonPCSpriteView(ctx, actor)
		}
	}
}

func (m *WorldMode) applyPetAction(ctx client.Context, action network.PetAction) {
	if action.Data < 5000 {
		emotionType := uint8(action.Data / 10)
		m.applyEmotionNotify(ctx, network.EmotionNotify{GID: action.ID, Type: emotionType})
		glog.Debugf("pet emotion actor=%d data=%d type=%d", action.ID, action.Data, emotionType)
		return
	}
	if ctx.Resources == nil {
		glog.Debugf("pet talk actor=%d data=%d ignored=no_resources", action.ID, action.Data)
		return
	}
	text, ok := ctx.Resources.PetTalk(action.Data)
	if !ok {
		glog.Debugf("pet talk actor=%d data=%d ignored=missing_text", action.ID, action.Data)
		return
	}
	m.applyPetTalk(ctx, action.ID, text, time.Now())
	glog.Debugf("pet talk actor=%d data=%d text=%q", action.ID, action.Data, text)
}

func (m *WorldMode) applyPetTalk(ctx client.Context, id uint32, text string, now time.Time) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	name := strings.TrimSpace(m.petProperty.Name)
	if ctx.World != nil {
		if actor, ok := ctx.World.Actors[id]; ok && strings.TrimSpace(actor.Name) != "" {
			name = strings.TrimSpace(actor.Name)
		}
	}
	if name == "" {
		name = "Pet"
	}
	chat := network.ChatMessage{GID: id, Text: fmt.Sprintf("%s : %s", name, text)}
	m.applySpeechBubble(ctx, chat, now)
	addConsoleMessage(&m.ui.console, ctx.Resources, chat)
}

func (m *WorldMode) maybeSendPetHuntingTalk(ctx client.Context, now time.Time) {
	if rand.IntN(10) >= 3 {
		return
	}
	m.sendPetTalkThrottled(ctx, petTalkHunting, now)
}

func (m *WorldMode) maybeSendPetDangerTalk(ctx client.Context, now time.Time) {
	m.sendPetTalkThrottled(ctx, petTalkDanger, now)
}

func (m *WorldMode) sendPetTalkThrottled(ctx client.Context, action int, now time.Time) {
	if !m.petLastTalk.IsZero() && m.petLastTalk.Add(petTalkCooldown).After(now) {
		return
	}
	if m.sendPetTalk(ctx, action) {
		m.petLastTalk = now
	}
}

func (m *WorldMode) sendPetTalk(ctx client.Context, action int) bool {
	if ctx.Network == nil || !m.hasPetProperty || m.petProperty.Relationship <= petTalkFriendlyThreshold {
		return false
	}
	job := m.petTalkJob(ctx)
	if job == 0 {
		return false
	}
	data := petTalkNumber(job, action, petHungryState(int(m.petOldFullness)))
	if err := ctx.Network.SendPetAct(data); err != nil {
		glog.Warnf("send pet talk failed action=%d data=%d: %v", action, data, err)
		return false
	}
	glog.Debugf("pet talk requested action=%d data=%d", action, data)
	return true
}

func (m *WorldMode) sendPetFeedEmotion(ctx client.Context) bool {
	if ctx.Network == nil || !m.hasPetProperty {
		return false
	}
	emotion := petEmotion(petHungryState(int(m.petOldFullness)), petFriendlyState(int(m.petProperty.Relationship)), petTalkFeeding)
	if emotion < 0 {
		return false
	}
	data := uint32(emotion*10 + 2)
	if err := ctx.Network.SendPetAct(data); err != nil {
		glog.Warnf("send pet feed emotion failed emotion=%d data=%d: %v", emotion, data, err)
		return false
	}
	glog.Debugf("pet feed emotion requested emotion=%d data=%d", emotion, data)
	return true
}

func (m *WorldMode) applyPetTalkParameterChange(ctx client.Context, change network.ParameterChange, previousHP int, previousBaseLevel int) {
	if ctx.Session == nil {
		return
	}
	now := time.Now()
	switch change.VarID {
	case network.StatusHP:
		hp := ctx.Session.Vitals.HP
		maxHP := ctx.Session.Vitals.MaxHP
		if maxHP <= 0 {
			maxHP = int(ctx.Session.Selected.MaxHP)
		}
		if hp <= 1 && previousHP > 1 {
			if m.sendPetTalk(ctx, petTalkDead) {
				m.petLastTalk = now
			}
			return
		}
		if maxHP > 0 && hp <= maxHP/4 {
			m.maybeSendPetDangerTalk(ctx, now)
		}
	case network.StatusBaseLevel:
		if ctx.Session.Progress.BaseLevel > previousBaseLevel {
			if m.sendPetTalk(ctx, petTalkLevelUp) {
				m.petLastTalk = now
			}
		}
	}
}

func (m *WorldMode) petTalkJob(ctx client.Context) int {
	if ctx.World != nil && m.petID != 0 {
		if actor, ok := ctx.World.Actors[m.petID]; ok && actor.Job >= 1000 {
			return int(actor.Job)
		}
	}
	if m.petProperty.Job >= 1000 {
		return int(m.petProperty.Job)
	}
	return 0
}

func petTalkNumber(job int, action int, hunger int) uint32 {
	if hunger < 0 {
		hunger = 0
	}
	if hunger > 99 {
		hunger = 99
	}
	if action < 0 {
		action = 0
	}
	return uint32(job*1000 + hunger*10 + action)
}

func petHungryState(fullness int) int {
	switch {
	case fullness > 90 && fullness <= 100:
		return 4
	case fullness > 75 && fullness <= 90:
		return 3
	case fullness > 25 && fullness <= 75:
		return 2
	case fullness > 10 && fullness <= 25:
		return 1
	default:
		return 0
	}
}

func petFriendlyState(relationship int) int {
	switch {
	case relationship > 900 && relationship <= 1000:
		return 4
	case relationship > 750 && relationship <= 900:
		return 3
	case relationship > 250 && relationship <= 750:
		return 2
	case relationship > 100 && relationship <= 250:
		return 1
	default:
		return 0
	}
}

func petEmotion(hunger int, friendly int, action int) int {
	if hunger < 0 || hunger >= len(petEmotionTable) {
		return -1
	}
	if friendly < 0 || friendly >= len(petEmotionTable[hunger]) {
		return -1
	}
	if action < 0 || action >= len(petEmotionTable[hunger][friendly]) {
		return -1
	}
	return petEmotionTable[hunger][friendly][action]
}

func (m *WorldMode) startPetPerformance(ctx client.Context, id uint32, data uint32) {
	if ctx.World == nil {
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	action := spriteActionNonPCPerf1
	switch data {
	case 2:
		action = spriteActionNonPCPerf2
	case 3:
		action = spriteActionNonPCPerf3
	case 4:
		action = spriteActionNonPCSpecial
	}
	now := time.Now()
	duration := m.actorActionDuration(ctx, actor, action, defaultAttackAnimationDuration)
	m.startActorAnimation(ctx, id, action, now, duration)
}

func (m *WorldMode) openPetContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || uiPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedPetTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths, m.petID)
	if !ok {
		return false
	}
	glog.Debugf("pet context target id=%d name=%q job=%d", actor.ID, actor.Name, actor.Job)
	m.ui.petContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY)
	return true
}

func (m *WorldMode) handlePetContextAction(ctx client.Context, action gameui.PetContextAction) {
	switch action.Kind {
	case gameui.PetContextActionInfo:
		m.openPetInfo(ctx)
	case gameui.PetContextActionFeed:
		m.openPetFeedConfirm(ctx)
	case gameui.PetContextActionPerformance:
		m.sendPetCommand(ctx, network.PetCommandPerformance)
	case gameui.PetContextActionBackToEgg:
		m.sendPetCommand(ctx, network.PetCommandBackToEgg)
	case gameui.PetContextActionUnequipAccessory:
		m.sendPetCommand(ctx, network.PetCommandUnequipAccessory)
	}
}

func (m *WorldMode) openPetInfo(ctx client.Context) {
	if m.hasPetProperty {
		m.ui.petInfoWindow.OpenInfo(ctx, m.petProperty)
		return
	}
	m.petInfoRequested = true
	m.sendPetCommand(ctx, network.PetCommandInfo)
}

func (m *WorldMode) openPetFeedConfirm(ctx client.Context) {
	m.ui.petConfirm.Open(ctx, "Feed Pet", "Are you sure you want to feed your pet?", func() {
		m.sendPetCommand(ctx, network.PetCommandFeed)
	}, nil)
}

func (m *WorldMode) sendPetCommand(ctx client.Context, command uint8) {
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Pet command failed: not connected.")
		return
	}
	if err := ctx.Network.SendPetCommand(command); err != nil {
		m.ui.console.AddErrorMessage("Pet command failed.")
		glog.Warnf("pet command failed command=%d: %v", command, err)
	}
}

func clickedPetTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time, petID uint32) (worldstate.Actor, bool) {
	if petID == 0 {
		return worldstate.Actor{}, false
	}
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || actor.ID != petID {
		return worldstate.Actor{}, false
	}
	return actor, true
}
