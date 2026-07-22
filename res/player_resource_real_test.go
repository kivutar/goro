package res

import (
	"testing"

	"github.com/kivutar/goro/db"
)

func TestPlayerBodyResourceRealWhenConfigured(t *testing.T) {
	job := 0
	sex := byte(1)

	manager := realDataManager(t)
	actSource, actData, ok := manager.ReadFirst(PlayerBodyResourceCandidates(job, sex, "act"))
	if !ok {
		t.Skipf("body act not found for job=%d sex=%d", job, sex)
	}
	sprSource, sprData, ok := manager.ReadFirst(PlayerBodyResourceCandidates(job, sex, "spr"))
	if !ok {
		t.Skipf("body spr not found for job=%d sex=%d", job, sex)
	}
	act, err := ParseACT(actData)
	if err != nil {
		t.Fatalf("parse %s: %v", actSource, err)
	}
	spr, err := ParseSPR(sprData)
	if err != nil {
		t.Fatalf("parse %s: %v", sprSource, err)
	}
	if len(act.Actions) == 0 || len(spr.Frames) == 0 {
		t.Fatalf("empty body resources: actions=%d frames=%d", len(act.Actions), len(spr.Frames))
	}
	t.Logf("body resources act=%s spr=%s actions=%d frames=%d", actSource, sprSource, len(act.Actions), len(spr.Frames))

	headActSource, headActData, ok := manager.ReadFirst(PlayerHeadResourceCandidates(job, 0, sex, "act"))
	if !ok {
		t.Skipf("head act not found for job=%d sex=%d", job, sex)
	}
	headSprSource, headSprData, ok := manager.ReadFirst(PlayerHeadResourceCandidates(job, 0, sex, "spr"))
	if !ok {
		t.Skipf("head spr not found for job=%d sex=%d", job, sex)
	}
	headAct, err := ParseACT(headActData)
	if err != nil {
		t.Fatalf("parse %s: %v", headActSource, err)
	}
	headSpr, err := ParseSPR(headSprData)
	if err != nil {
		t.Fatalf("parse %s: %v", headSprSource, err)
	}
	if len(headAct.Actions) == 0 || len(headSpr.Frames) == 0 {
		t.Fatalf("empty head resources: actions=%d frames=%d", len(headAct.Actions), len(headSpr.Frames))
	}
	t.Logf("head resources act=%s spr=%s actions=%d frames=%d", headActSource, headSprSource, len(headAct.Actions), len(headSpr.Frames))
}

func TestPlayerMageEquippedRodOverlayRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	job := 2
	sex := byte(0)
	weapon := manager.PlayerWeaponViewID(1607)
	if weapon != 10 {
		t.Fatalf("mage weapon view id for item 1607 = %d, want reference client class num 10", weapon)
	}

	actSource, actData, ok := manager.ReadFirst(PlayerWeaponOverlayResourceCandidates(job, sex, weapon, false, "act"))
	if !ok {
		t.Fatalf("mage rod act not found candidates=%q", PlayerWeaponOverlayResourceCandidates(job, sex, weapon, false, "act"))
	}
	sprSource, sprData, ok := manager.ReadFirst(PlayerWeaponOverlayResourceCandidates(job, sex, weapon, false, "spr"))
	if !ok {
		t.Fatalf("mage rod spr not found candidates=%q", PlayerWeaponOverlayResourceCandidates(job, sex, weapon, false, "spr"))
	}
	act, err := ParseACT(actData)
	if err != nil {
		t.Fatalf("parse %s: %v", actSource, err)
	}
	spr, err := ParseSPR(sprData)
	if err != nil {
		t.Fatalf("parse %s: %v", sprSource, err)
	}
	t.Logf("mage rod resources act=%s spr=%s actions=%d frames=%d", actSource, sprSource, len(act.Actions), len(spr.Frames))
	for _, actionIndex := range []int{40, 80, 88} {
		if actionIndex >= len(act.Actions) {
			t.Fatalf("mage rod action %d missing; actions=%d", actionIndex, len(act.Actions))
		}
		action := act.Actions[actionIndex]
		visible := 0
		for _, anim := range action.Animations {
			for _, layer := range anim.Layers {
				if layer.Index >= 0 {
					visible++
					break
				}
			}
		}
		t.Logf("mage rod action=%d animations=%d visible=%d delay=%.1f", actionIndex, len(action.Animations), visible, action.DelayMS)
		if actionIndex == 80 && visible == 0 {
			t.Fatalf("mage rod attack action has no visible frames")
		}
	}
}

func TestMercenaryWeaponOverlayResourceRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	for _, tc := range []struct {
		name        string
		resource    string
		weapon      int
		secondLayer bool
	}{
		{name: "archer bow", resource: `여\활용병`, weapon: db.WeaponBow},
		{name: "lancer spear", resource: `남\창용병`, weapon: db.WeaponSpear},
		{name: "lancer spear light", resource: `남\창용병`, weapon: db.WeaponSpear, secondLayer: true},
		{name: "sword sword", resource: `남\검용병`, weapon: db.WeaponSword},
		{name: "sword sword light", resource: `남\검용병`, weapon: db.WeaponSword, secondLayer: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actCandidates := MercenaryWeaponOverlayResourceCandidates(tc.resource, tc.weapon, tc.secondLayer, "act")
			sprCandidates := MercenaryWeaponOverlayResourceCandidates(tc.resource, tc.weapon, tc.secondLayer, "spr")
			actSource, actData, ok := manager.ReadFirst(actCandidates)
			if !ok {
				t.Fatalf("mercenary weapon act not found candidates=%q", actCandidates)
			}
			sprSource, sprData, ok := manager.ReadFirst(sprCandidates)
			if !ok {
				t.Fatalf("mercenary weapon spr not found candidates=%q", sprCandidates)
			}
			act, err := ParseACT(actData)
			if err != nil {
				t.Fatalf("parse %s: %v", actSource, err)
			}
			spr, err := ParseSPR(sprData)
			if err != nil {
				t.Fatalf("parse %s: %v", sprSource, err)
			}
			if len(act.Actions) == 0 || len(spr.Frames) == 0 {
				t.Fatalf("empty mercenary weapon resources act=%s spr=%s actions=%d frames=%d", actSource, sprSource, len(act.Actions), len(spr.Frames))
			}
		})
	}
}
