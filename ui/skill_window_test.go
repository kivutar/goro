package ui

import (
	"image"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestCanIncreaseSkillRequiresPointsAndFlag(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 1}}
	if !canIncreaseSkill(s, session.Skill{ID: 1, Upgradable: true}) {
		t.Fatal("expected skill to be increasable")
	}
	if canIncreaseSkill(s, session.Skill{ID: 1}) {
		t.Fatal("skill without upgradable flag should not increase")
	}
	s.Skills.Points = 0
	if canIncreaseSkill(s, session.Skill{ID: 1, Upgradable: true}) {
		t.Fatal("skill without points should not increase")
	}
}

func TestSkillWindowHidesUnavailableLevelUpButton(t *testing.T) {
	window := &SkillWindow{}
	cell := window.skillTableCell(
		Context{Session: &session.Session{}},
		nil,
		session.Skill{ID: db.SkillSMBash, Level: 1, Upgradable: true},
		rotheme.TableViewCellContext{Column: rotheme.TableViewColumn{Key: "levelup"}},
	)
	if !cell.Hidden || cell.HasIconButton {
		t.Fatalf("unavailable skill level-up cell = %+v, want hidden", cell)
	}
}

func TestSkillWindowFooterRequiresPendingChanges(t *testing.T) {
	for _, mode := range []string{"list", "grid"} {
		t.Run(mode, func(t *testing.T) {
			window := &SkillWindow{gridMode: mode == "grid"}
			ctx := Context{Session: &session.Session{Skills: session.Skills{Points: 3}}}
			for _, phase := range []string{"initial", "staged", "cleared"} {
				t.Run(phase, func(t *testing.T) {
					switch phase {
					case "staged":
						window.stageSkill(db.SkillNVBasic)
					case "cleared":
						window.clearPending()
					}
					children := window.widgetTree(ctx, nil).Children()
					footer := children[len(children)-1]
					buttons := 0
					var visit func(widget.Widget)
					visit = func(node widget.Widget) {
						if btn, ok := node.(*button.Widget); ok {
							buttons++
							// IsFocusable includes the button's configured disabled state.
							if got, want := btn.IsFocusable(), phase == "staged"; got != want {
								t.Errorf("footer button %d enabled = %t, want %t", buttons, got, want)
							}
						}
						for _, child := range node.Children() {
							visit(child)
						}
					}
					visit(footer)
					if buttons != 2 {
						t.Fatalf("footer buttons = %d, want Reset and Confirm", buttons)
					}
				})
			}
		})
	}
}

func TestSkillWindowCanStageSkillHonorsMaxLevel(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 3}}
	window := &SkillWindow{}
	skill := session.Skill{ID: db.SkillSMBash, Level: 9, Upgradable: true}
	if !window.canStageSkill(s, skill) {
		t.Fatal("expected level 9/10 skill to allow one staged level")
	}
	window.stageSkill(skill.ID)
	if window.canStageSkill(s, skill) {
		t.Fatal("level 9/10 skill should not allow staging past level 10")
	}
	if canIncreaseSkill(s, session.Skill{ID: db.SkillSMBash, Level: 10, Upgradable: true}) {
		t.Fatal("max-level skill should not increase")
	}
}

func TestSkillWindowCanStageSkillUsesDBMaxBeforeResourceMax(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 3}}
	window := &SkillWindow{}
	skill := session.Skill{ID: db.SkillHTBlitzbeat, Level: 4, MaxLevel: 10, Upgradable: true}
	if !window.canStageSkill(s, skill) {
		t.Fatal("expected level 4/5 blitz beat to allow one staged level")
	}
	window.stageSkill(skill.ID)
	if window.canStageSkill(s, skill) {
		t.Fatal("blitz beat should not stage past db max level 5")
	}
	if canIncreaseSkill(s, session.Skill{ID: db.SkillHTBlitzbeat, Level: 5, MaxLevel: 10, Upgradable: true}) {
		t.Fatal("db max should keep blitz beat from increasing at level 5")
	}
}

func TestSkillWindowCanStageSkillWithoutKnownMaxAllowsAvailablePoints(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 3}}
	window := &SkillWindow{}
	skill := session.Skill{ID: 999, Level: 1, Upgradable: true}
	for i := 0; i < s.Skills.Points; i++ {
		if !window.canStageSkill(s, skill) {
			t.Fatalf("expected unknown max skill to allow staged level %d", i+1)
		}
		window.stageSkill(skill.ID)
	}
	if window.canStageSkill(s, skill) {
		t.Fatal("unknown max skill should not allow staging past available points")
	}
}

func TestSkillWindowShowsPendingUnlockedSkillBeforeConfirm(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSwordman},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMBash, Level: 4, MaxLevel: 10, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}

	if containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillSMMagnum) {
		t.Fatal("magnum break should not be unlocked before bash reaches level 5")
	}
	window.stageSkill(db.SkillSMBash)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillSMMagnum) {
		t.Fatal("magnum break should be unlocked after staged bash level satisfies prerequisites")
	}
}

func TestSkillWindowShowsSuperNoviceThunderstorm(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSuperNovice},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillMGLightningbolt, Level: 3, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}

	if containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillMGThunderstorm) {
		t.Fatal("thunderstorm should not be unlocked before lightning bolt reaches level 4")
	}
	window.stageSkill(db.SkillMGLightningbolt)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillMGThunderstorm) {
		t.Fatal("super novice should unlock thunderstorm after staged lightning bolt level satisfies prerequisites")
	}
}

func TestSkillWindowShowsWizardUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobWizard},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillMGStonecurse, Level: 1, Upgradable: true},
				{ID: db.SkillMGColdbolt, Level: 1, Upgradable: true},
				{ID: db.SkillMGLightningbolt, Level: 1, Upgradable: true},
				{ID: db.SkillMGNapalmbeat, Level: 1, Upgradable: true},
				{ID: db.SkillMGSight, Level: 1, Upgradable: true},
				{ID: db.SkillMGThunderstorm, Level: 1, Upgradable: true},
				{ID: db.SkillMGFrostdiver, Level: 1, Upgradable: true},
				{ID: db.SkillMGFirewall, Level: 1, Upgradable: true},
				{ID: db.SkillWZJupitel, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillWZFirepillar,
		db.SkillWZSightrasher,
		db.SkillWZJupitel,
		db.SkillWZWaterball,
		db.SkillWZIcewall,
		db.SkillWZEarthspike,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("wizard tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillWZStormgust) {
		t.Fatal("storm gust should not be unlocked before jupitel thunder reaches level 3")
	}
	window.stageSkill(db.SkillWZJupitel)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillWZStormgust) {
		t.Fatal("storm gust should be unlocked after staged jupitel thunder level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsHighWizardUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobWizardH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillMGSrecovery, Level: 5, Upgradable: true},
				{ID: db.SkillMGSoulstrike, Level: 7, Upgradable: true},
				{ID: db.SkillMGNapalmbeat, Level: 5, Upgradable: true},
				{ID: db.SkillWZEstimation, Level: 1, Upgradable: true},
				{ID: db.SkillWZIcewall, Level: 1, Upgradable: true},
				{ID: db.SkillWZQuagmire, Level: 1, Upgradable: true},
				{ID: db.SkillHWMagiccrasher, Level: 1, Upgradable: true},
				{ID: db.SkillHWMagicpower, Level: 9, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillHWSouldrain,
		db.SkillHWMagiccrasher,
		db.SkillHWNapalmvulcan,
		db.SkillHWGanbantein,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("high wizard tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillHWGravitation) {
		t.Fatal("gravitation should not be unlocked before magic power reaches level 10")
	}
	window.stageSkill(db.SkillHWMagicpower)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillHWGravitation) {
		t.Fatal("gravitation should be unlocked after staged magic power level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsSageUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSage},
		Skills: session.Skills{
			Points: 3,
			List: []session.Skill{
				{ID: db.SkillMGFirebolt, Level: 1, Upgradable: true},
				{ID: db.SkillMGColdbolt, Level: 1, Upgradable: true},
				{ID: db.SkillMGLightningbolt, Level: 1, Upgradable: true},
				{ID: db.SkillMGStonecurse, Level: 1, Upgradable: true},
				{ID: db.SkillSAAdvancedbook, Level: 5, Upgradable: true},
				{ID: db.SkillSACastcancel, Level: 1, Upgradable: true},
				{ID: db.SkillSAMagicrod, Level: 1, Upgradable: true},
				{ID: db.SkillSAFlamelauncher, Level: 2, Upgradable: true},
				{ID: db.SkillSAFrostweapon, Level: 2, Upgradable: true},
				{ID: db.SkillSALightningloader, Level: 2, Upgradable: true},
				{ID: db.SkillSADeluge, Level: 3, Upgradable: true},
				{ID: db.SkillSAViolentgale, Level: 3, Upgradable: true},
				{ID: db.SkillSAVolcano, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillSAFlamelauncher,
		db.SkillSAFrostweapon,
		db.SkillSALightningloader,
		db.SkillSASeismicweapon,
		db.SkillSAFreecast,
		db.SkillSASpellbreaker,
		db.SkillSADeluge,
		db.SkillSAViolentgale,
		db.SkillSAVolcano,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("sage tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillWZEarthspike) {
		t.Fatal("earth spike should not be unlocked for Sage before seismic weapon reaches level 1")
	}
	window.stageSkill(db.SkillSASeismicweapon)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillWZEarthspike) {
		t.Fatalf("earth spike should be unlocked after staged seismic weapon satisfies robr Sage prerequisite: %v", skills)
	}
	if containsUnlockedSkill(skills, db.SkillWZHeavendrive) {
		t.Fatal("heaven's drive should not be unlocked for Sage before earth spike reaches level 1")
	}
	window.stageSkill(db.SkillWZEarthspike)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillWZHeavendrive) {
		t.Fatal("heaven's drive should be unlocked after staged earth spike satisfies robr Sage prerequisite")
	}
	if containsUnlockedSkill(skills, db.SkillSALandprotector) {
		t.Fatal("land protector should not be unlocked before volcano reaches level 3")
	}
	window.stageSkill(db.SkillSAVolcano)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillSALandprotector) {
		t.Fatal("land protector should be unlocked after staged volcano level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsProfessorUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSageH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillMGSrecovery, Level: 3, Upgradable: true},
				{ID: db.SkillSAAdvancedbook, Level: 5, Upgradable: true},
				{ID: db.SkillSACastcancel, Level: 5, Upgradable: true},
				{ID: db.SkillSAMagicrod, Level: 3, Upgradable: true},
				{ID: db.SkillSASpellbreaker, Level: 2, Upgradable: true},
				{ID: db.SkillSAFreecast, Level: 5, Upgradable: true},
				{ID: db.SkillSAAutospell, Level: 1, Upgradable: true},
				{ID: db.SkillSADragonology, Level: 4, Upgradable: true},
				{ID: db.SkillSADeluge, Level: 2, Upgradable: true},
				{ID: db.SkillSAViolentgale, Level: 2, Upgradable: true},
				{ID: db.SkillSADispell, Level: 3, Upgradable: true},
				{ID: db.SkillPFSoulburn, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillPFSpiderweb,
		db.SkillPFSoulchange,
		db.SkillPFFogwall,
		db.SkillPFHpconversion,
		db.SkillPFDoublecasting,
		db.SkillPFMemorize,
		db.SkillPFSoulburn,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("professor tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillPFMindbreaker) {
		t.Fatal("mind breaker should not be unlocked before soul burn reaches level 2")
	}
	window.stageSkill(db.SkillPFSoulburn)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillPFMindbreaker) {
		t.Fatal("mind breaker should be unlocked after staged soul burn level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsKnightUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobKnight},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMTwohand, Level: 1, Upgradable: true},
				{ID: db.SkillSMEndure, Level: 1, Upgradable: true},
				{ID: db.SkillKNSpearmastery, Level: 1, Upgradable: true},
				{ID: db.SkillKNRiding, Level: 1, Upgradable: true},
				{ID: db.SkillKNPierce, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillKNTwohandquicken,
		db.SkillKNAutocounter,
		db.SkillKNRiding,
		db.SkillKNPierce,
		db.SkillKNCavaliermastery,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("knight tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillKNSpearboomerang) {
		t.Fatal("spear boomerang should not be unlocked before pierce reaches level 3")
	}
	window.stageSkill(db.SkillKNPierce)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillKNSpearboomerang) {
		t.Fatal("spear boomerang should be unlocked after staged pierce level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsLordKnightUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobKnightH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMMagnum, Level: 5, Upgradable: true},
				{ID: db.SkillSMTwohand, Level: 10, Upgradable: true},
				{ID: db.SkillSMProvoke, Level: 5, Upgradable: true},
				{ID: db.SkillSMRecovery, Level: 10, Upgradable: true},
				{ID: db.SkillSMEndure, Level: 3, Upgradable: true},
				{ID: db.SkillKNTwohandquicken, Level: 3, Upgradable: true},
				{ID: db.SkillKNSpearmastery, Level: 9, Upgradable: true},
				{ID: db.SkillKNPierce, Level: 5, Upgradable: true},
				{ID: db.SkillKNRiding, Level: 1, Upgradable: true},
				{ID: db.SkillKNSpearstab, Level: 5, Upgradable: true},
				{ID: db.SkillKNCavaliermastery, Level: 3, Upgradable: true},
				{ID: db.SkillLKHeadcrush, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillLKAurablade,
		db.SkillLKParrying,
		db.SkillLKConcentration,
		db.SkillLKHeadcrush,
		db.SkillLKSpiralpierce,
		db.SkillLKTensionrelax,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("lord knight tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillLKJointbeat) {
		t.Fatal("joint beat should not be unlocked before head crush reaches level 3")
	}
	window.stageSkill(db.SkillLKHeadcrush)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillLKJointbeat) {
		t.Fatal("joint beat should be unlocked after staged head crush level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsCrusaderUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobCrusader},
		Skills: session.Skills{
			Points: 2,
			List: []session.Skill{
				{ID: db.SkillCRTrust, Level: 6, Upgradable: true},
				{ID: db.SkillCRAutoguard, Level: 5, Upgradable: true},
				{ID: db.SkillCRShieldcharge, Level: 3, Upgradable: true},
				{ID: db.SkillCRShieldboomerang, Level: 2, Upgradable: true},
				{ID: db.SkillKNSpearmastery, Level: 10, Upgradable: true},
				{ID: db.SkillALCure, Level: 1, Upgradable: true},
				{ID: db.SkillALDp, Level: 5, Upgradable: true},
				{ID: db.SkillALHeal, Level: 5, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillALCure,
		db.SkillALDp,
		db.SkillCRShieldcharge,
		db.SkillCRShieldboomerang,
		db.SkillCRDefender,
		db.SkillCRProvidence,
		db.SkillCRSpearquicken,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("crusader tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillCRReflectshield) {
		t.Fatal("reflect shield should not be unlocked before shield boomerang reaches level 3")
	}
	if containsUnlockedSkill(skills, db.SkillCRHolycross) {
		t.Fatal("holy cross should not be unlocked before faith reaches level 7")
	}
	window.stageSkill(db.SkillCRShieldboomerang)
	window.stageSkill(db.SkillCRTrust)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillCRReflectshield) {
		t.Fatal("reflect shield should be unlocked after staged shield boomerang level satisfies robr prerequisite")
	}
	if !containsUnlockedSkill(skills, db.SkillCRHolycross) {
		t.Fatal("holy cross should be unlocked after staged faith level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsPaladinUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobCrusaderH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMEndure, Level: 5, Upgradable: true},
				{ID: db.SkillCRTrust, Level: 8, Upgradable: true},
				{ID: db.SkillCRShieldcharge, Level: 2, Upgradable: true},
				{ID: db.SkillCRShieldboomerang, Level: 5, Upgradable: true},
				{ID: db.SkillCRDevotion, Level: 2, Upgradable: true},
				{ID: db.SkillALDp, Level: 3, Upgradable: true},
				{ID: db.SkillALDemonbane, Level: 5, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillPaPressure,
		db.SkillPaShieldchain,
		db.SkillPaGospel,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("paladin tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillPaSacrifice) {
		t.Fatal("martyr's reckoning should not be unlocked before devotion reaches level 3")
	}
	window.stageSkill(db.SkillCRDevotion)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillPaSacrifice) {
		t.Fatal("martyr's reckoning should be unlocked after staged devotion level satisfies robr prerequisite")
	}
}

func TestSkillWindowRequiresFaithFiveForMartyrsReckoning(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobCrusaderH},
		Skills: session.Skills{List: []session.Skill{
			{ID: db.SkillSMEndure, Level: 1, Upgradable: true},
			{ID: db.SkillCRTrust, Level: 4, Upgradable: true},
			{ID: db.SkillCRDevotion, Level: 3, Upgradable: true},
		}},
	}
	window := &SkillWindow{}
	if containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillPaSacrifice) {
		t.Fatal("martyr's reckoning should not be unlocked before Faith reaches level 5")
	}
	s.Skills.List[1].Level = 5
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillPaSacrifice) {
		t.Fatal("martyr's reckoning should be unlocked when Endure, Faith, and Devotion requirements are met")
	}
}

func TestSkillWindowShowsPriestUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobPriest},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillALHeal, Level: 1, Upgradable: true},
				{ID: db.SkillALAngelus, Level: 2, Upgradable: true},
				{ID: db.SkillMGSrecovery, Level: 4, Upgradable: true},
				{ID: db.SkillPRStrecovery, Level: 1, Upgradable: true},
				{ID: db.SkillPRMagnificat, Level: 3, Upgradable: true},
				{ID: db.SkillPRKyrie, Level: 4, Upgradable: true},
				{ID: db.SkillPRSanctuary, Level: 3, Upgradable: true},
				{ID: db.SkillPRAspersio, Level: 3, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillPRSanctuary,
		db.SkillPRKyrie,
		db.SkillPRGloria,
		db.SkillALLResurrection,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("priest tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillMGSafetywall) {
		t.Fatal("priest safety wall should not be unlocked before aspersio reaches level 4")
	}
	window.stageSkill(db.SkillPRAspersio)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillMGSafetywall) {
		t.Fatal("priest safety wall should be unlocked after staged aspersio satisfies robr priest prerequisite")
	}
}

func TestSkillWindowShowsHighPriestUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobPriestH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillALAngelus, Level: 1, Upgradable: true},
				{ID: db.SkillALDemonbane, Level: 10, Upgradable: true},
				{ID: db.SkillMGSrecovery, Level: 5, Upgradable: true},
				{ID: db.SkillPRImpositio, Level: 3, Upgradable: true},
				{ID: db.SkillPRGloria, Level: 2, Upgradable: true},
				{ID: db.SkillPRKyrie, Level: 3, Upgradable: true},
				{ID: db.SkillPRMacemastery, Level: 10, Upgradable: true},
				{ID: db.SkillPRLexdivina, Level: 5, Upgradable: true},
				{ID: db.SkillPRAspersio, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillHPAssumptio,
		db.SkillHPBasilica,
		db.SkillHPManarecharge,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("high priest tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillHPMeditatio) {
		t.Fatal("meditatio should not be unlocked before aspersio reaches level 3")
	}
	window.stageSkill(db.SkillPRAspersio)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillHPMeditatio) {
		t.Fatal("meditatio should be unlocked after staged aspersio level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsMonkUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobMonk},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillALDemonbane, Level: 10, Upgradable: true},
				{ID: db.SkillALDp, Level: 10, Upgradable: true},
				{ID: db.SkillMOIronhand, Level: 5, Upgradable: true},
				{ID: db.SkillMOCallspirits, Level: 4, Upgradable: true},
				{ID: db.SkillMODodge, Level: 5, Upgradable: true},
				{ID: db.SkillMOTripleattack, Level: 5, Upgradable: true},
				{ID: db.SkillMOChaincombo, Level: 3, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillMOIronhand,
		db.SkillMOCallspirits,
		db.SkillMODodge,
		db.SkillMOTripleattack,
		db.SkillMOBladestop,
		db.SkillMOCombofinish,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("monk tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillMOAbsorbspirits) {
		t.Fatal("absorb spirits should not be unlocked before call spirits reaches level 5")
	}
	if containsUnlockedSkill(skills, db.SkillMOInvestigate) {
		t.Fatal("investigate should not be unlocked before call spirits reaches level 5")
	}
	window.stageSkill(db.SkillMOCallspirits)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillMOAbsorbspirits) {
		t.Fatal("absorb spirits should be unlocked after staged call spirits level satisfies robr prerequisite")
	}
	if !containsUnlockedSkill(skills, db.SkillMOInvestigate) {
		t.Fatal("investigate should be unlocked after staged call spirits level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsGunslingerUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobGunslinger},
		Skills: session.Skills{
			Points: 2,
			List: []session.Skill{
				{ID: db.SkillGSGlittering, Level: 3, Upgradable: true},
				{ID: db.SkillGSSingleaction, Level: 4, Upgradable: true},
				{ID: db.SkillGSSnakeeye, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillGSFling,
		db.SkillGSTripleaction,
		db.SkillGSIncreasing,
		db.SkillGSMagicalbullet,
		db.SkillGSCracker,
		db.SkillGSChainaction,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("Gunslinger tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	for _, skillID := range []uint16{
		db.SkillGSMadnesscancel,
		db.SkillGSAdjustment,
		db.SkillGSTracking,
		db.SkillGSDust,
		db.SkillGSSpreadattack,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("skill %d should not be unlocked before its prerequisite reaches the robr level", skillID)
		}
	}
	window.stageSkill(db.SkillGSGlittering)
	window.stageSkill(db.SkillGSSingleaction)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillGSMadnesscancel,
		db.SkillGSAdjustment,
		db.SkillGSTracking,
		db.SkillGSDust,
		db.SkillGSSpreadattack,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("staged prerequisite did not expose Gunslinger skill %d: %v", skillID, skills)
		}
	}
}

func TestSkillWindowShowsNinjaUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobNinja},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillNJTobidougu, Level: 5, Upgradable: true},
				{ID: db.SkillNJTatamigaeshi, Level: 1, Upgradable: true},
				{ID: db.SkillNJNinpou, Level: 1, Upgradable: true},
				{ID: db.SkillNJSyuriken, Level: 4, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillNJShadowjump,
		db.SkillNJKouenka,
		db.SkillNJHyousensou,
		db.SkillNJHuujin,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("Ninja tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillNJKunai) {
		t.Fatal("kunai should not be unlocked before throw shuriken reaches level 5")
	}
	window.stageSkill(db.SkillNJSyuriken)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillNJKunai) {
		t.Fatal("kunai should be unlocked after staged throw shuriken level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsChampionUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobMonkH},
		Skills: session.Skills{
			Points: 2,
			List: []session.Skill{
				{ID: db.SkillMOIronhand, Level: 7, Upgradable: true},
				{ID: db.SkillMOCallspirits, Level: 5, Upgradable: true},
				{ID: db.SkillMOExplosionspirits, Level: 4, Upgradable: true},
				{ID: db.SkillMOTripleattack, Level: 5, Upgradable: true},
				{ID: db.SkillMOCombofinish, Level: 3, Upgradable: true},
				{ID: db.SkillChTigerfist, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillChPalmstrike,
		db.SkillChTigerfist,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("champion tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillChSoulcollect) {
		t.Fatal("zen should not be unlocked before fury reaches level 5")
	}
	if containsUnlockedSkill(skills, db.SkillChChaincrush) {
		t.Fatal("chain crush combo should not be unlocked before tiger fist reaches level 2")
	}
	window.stageSkill(db.SkillMOExplosionspirits)
	window.stageSkill(db.SkillChTigerfist)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillChSoulcollect) {
		t.Fatal("zen should be unlocked after staged fury level satisfies robr prerequisite")
	}
	if !containsUnlockedSkill(skills, db.SkillChChaincrush) {
		t.Fatal("chain crush combo should be unlocked after staged tiger fist level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsHunterUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobHunter},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillACConcentration, Level: 1, Upgradable: true},
				{ID: db.SkillHTSkidtrap, Level: 1, Upgradable: true},
				{ID: db.SkillHTLandmine, Level: 1, Upgradable: true},
				{ID: db.SkillHTFalcon, Level: 1, Upgradable: true},
				{ID: db.SkillHTFlasher, Level: 1, Upgradable: true},
				{ID: db.SkillHTAnklesnare, Level: 1, Upgradable: true},
				{ID: db.SkillHTRemovetrap, Level: 1, Upgradable: true},
				{ID: db.SkillHTSandman, Level: 1, Upgradable: true},
				{ID: db.SkillHTFreezingtrap, Level: 0, Upgradable: true},
				{ID: db.SkillHTShockwave, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillHTFlasher,
		db.SkillHTAnklesnare,
		db.SkillHTShockwave,
		db.SkillHTSpringtrap,
		db.SkillHTDetecting,
		db.SkillHTTalkiebox,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("hunter tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillHTBlastmine) {
		t.Fatal("blast mine should not be unlocked before freezing trap reaches level 1")
	}
	window.stageSkill(db.SkillHTFreezingtrap)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillHTBlastmine) {
		t.Fatal("blast mine should be unlocked after staged freezing trap level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsSniperUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobHunterH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillACOwl, Level: 10, Upgradable: true},
				{ID: db.SkillACVulture, Level: 10, Upgradable: true},
				{ID: db.SkillACConcentration, Level: 10, Upgradable: true},
				{ID: db.SkillACDouble, Level: 5, Upgradable: true},
				{ID: db.SkillHTFalcon, Level: 1, Upgradable: true},
				{ID: db.SkillHTBlitzbeat, Level: 5, Upgradable: true},
				{ID: db.SkillHTSteelcrow, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillSNSharpshooting,
		db.SkillSNSight,
		db.SkillSNWindwalk,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("sniper tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillSNFalconassault) {
		t.Fatal("falcon assault should not be unlocked before steel crow reaches level 3")
	}
	window.stageSkill(db.SkillHTSteelcrow)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillSNFalconassault) {
		t.Fatal("falcon assault should be unlocked after staged steel crow level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsBardUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobBard},
		Skills: session.Skills{
			Points: 3,
			List: []session.Skill{
				{ID: db.SkillBDAdaptation, Level: 1, Upgradable: true},
				{ID: db.SkillBDEncore, Level: 1, Upgradable: true},
				{ID: db.SkillBaMusicallesson, Level: 2, Upgradable: true},
				{ID: db.SkillBaDissonance, Level: 2, Upgradable: true},
				{ID: db.SkillBaWhistle, Level: 9, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillBaFrostjoke) {
		t.Fatalf("bard tree did not expose frost joke from encore: %v", skills)
	}
	for _, skillID := range []uint16{
		db.SkillBaMusicalstrike,
		db.SkillBaAssassincross,
		db.SkillBaPoembragi,
		db.SkillBaAppleidun,
		db.SkillBDLullaby,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("bard skill %d should not be unlocked before staged prerequisite: %v", skillID, skills)
		}
	}
	window.stageSkill(db.SkillBaMusicallesson)
	window.stageSkill(db.SkillBaDissonance)
	window.stageSkill(db.SkillBaWhistle)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillBaMusicalstrike,
		db.SkillBaAssassincross,
		db.SkillBaPoembragi,
		db.SkillBaAppleidun,
		db.SkillBDLullaby,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("bard staged prerequisites did not expose skill %d: %v", skillID, skills)
		}
	}
}

func TestSkillWindowShowsDancerUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobDancer},
		Skills: session.Skills{
			Points: 3,
			List: []session.Skill{
				{ID: db.SkillBDAdaptation, Level: 1, Upgradable: true},
				{ID: db.SkillBDEncore, Level: 1, Upgradable: true},
				{ID: db.SkillDCDancinglesson, Level: 2, Upgradable: true},
				{ID: db.SkillDCUglydance, Level: 2, Upgradable: true},
				{ID: db.SkillDCHumming, Level: 9, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillDCScream) {
		t.Fatalf("dancer tree did not expose scream from encore: %v", skills)
	}
	for _, skillID := range []uint16{
		db.SkillDCThrowarrow,
		db.SkillDCDontforgetme,
		db.SkillDCFortunekiss,
		db.SkillDCServiceforyou,
		db.SkillBDLullaby,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("dancer skill %d should not be unlocked before staged prerequisite: %v", skillID, skills)
		}
	}
	window.stageSkill(db.SkillDCDancinglesson)
	window.stageSkill(db.SkillDCUglydance)
	window.stageSkill(db.SkillDCHumming)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillDCThrowarrow,
		db.SkillDCDontforgetme,
		db.SkillDCFortunekiss,
		db.SkillDCServiceforyou,
		db.SkillBDLullaby,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("dancer staged prerequisites did not expose skill %d: %v", skillID, skills)
		}
	}
}

func TestSkillWindowShowsClownUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobBardH},
		Skills: session.Skills{
			Points: 3,
			List: []session.Skill{
				{ID: db.SkillACDouble, Level: 5, Upgradable: true},
				{ID: db.SkillACShower, Level: 5, Upgradable: true},
				{ID: db.SkillACConcentration, Level: 10, Upgradable: true},
				{ID: db.SkillBaMusicallesson, Level: 9, Upgradable: true},
				{ID: db.SkillBaMusicalstrike, Level: 1, Upgradable: true},
				{ID: db.SkillBaDissonance, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillCGArrowvulcan,
		db.SkillCGMoonlit,
		db.SkillCGMarionette,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("clown tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	for _, skillID := range []uint16{
		db.SkillCGHermode,
		db.SkillCGTarotcard,
		db.SkillCGLongingfreedom,
		db.SkillCGSpecialsinger,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("clown skill %d should not be unlocked before staged prerequisite: %v", skillID, skills)
		}
	}
	window.stageSkill(db.SkillBaMusicallesson)
	window.stageSkill(db.SkillBaDissonance)
	window.stageSkill(db.SkillCGMarionette)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillCGHermode,
		db.SkillCGTarotcard,
		db.SkillCGLongingfreedom,
		db.SkillCGSpecialsinger,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("clown staged prerequisites did not expose skill %d: %v", skillID, skills)
		}
	}
}

func TestSkillWindowShowsGypsyUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobDancerH},
		Skills: session.Skills{
			Points: 3,
			List: []session.Skill{
				{ID: db.SkillACDouble, Level: 5, Upgradable: true},
				{ID: db.SkillACShower, Level: 5, Upgradable: true},
				{ID: db.SkillACConcentration, Level: 10, Upgradable: true},
				{ID: db.SkillDCDancinglesson, Level: 9, Upgradable: true},
				{ID: db.SkillDCThrowarrow, Level: 1, Upgradable: true},
				{ID: db.SkillDCUglydance, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillCGArrowvulcan,
		db.SkillCGMoonlit,
		db.SkillCGMarionette,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("gypsy tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	for _, skillID := range []uint16{
		db.SkillCGHermode,
		db.SkillCGTarotcard,
		db.SkillCGLongingfreedom,
		db.SkillCGSpecialsinger,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("gypsy skill %d should not be unlocked before staged prerequisite: %v", skillID, skills)
		}
	}
	window.stageSkill(db.SkillDCDancinglesson)
	window.stageSkill(db.SkillDCUglydance)
	window.stageSkill(db.SkillCGMarionette)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillCGHermode,
		db.SkillCGTarotcard,
		db.SkillCGLongingfreedom,
		db.SkillCGSpecialsinger,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("gypsy staged prerequisites did not expose skill %d: %v", skillID, skills)
		}
	}
}

func TestSkillWindowShowsAssassinUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobAssassin},
		Skills: session.Skills{
			Points: 2,
			List: []session.Skill{
				{ID: db.SkillTFHiding, Level: 2, Upgradable: true},
				{ID: db.SkillTFPoison, Level: 1, Upgradable: true},
				{ID: db.SkillASRight, Level: 2, Upgradable: true},
				{ID: db.SkillASKatar, Level: 4, Upgradable: true},
				{ID: db.SkillASCloaking, Level: 2, Upgradable: true},
				{ID: db.SkillASSonicblow, Level: 4, Upgradable: true},
				{ID: db.SkillASEnchantpoison, Level: 5, Upgradable: true},
				{ID: db.SkillASVenomdust, Level: 5, Upgradable: true},
				{ID: db.SkillASPoisonreact, Level: 4, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillASCloaking,
		db.SkillASEnchantpoison,
		db.SkillASLeft,
		db.SkillASSonicblow,
		db.SkillASVenomdust,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("assassin tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillASGrimtooth) {
		t.Fatal("grimtooth should not be unlocked before sonic blow reaches level 5")
	}
	if containsUnlockedSkill(skills, db.SkillASSplasher) {
		t.Fatal("venom splasher should not be unlocked before poison react reaches level 5")
	}
	window.stageSkill(db.SkillASSonicblow)
	window.stageSkill(db.SkillASPoisonreact)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillASGrimtooth) {
		t.Fatal("grimtooth should be unlocked after staged sonic blow level satisfies robr prerequisite")
	}
	if !containsUnlockedSkill(skills, db.SkillASSplasher) {
		t.Fatal("venom splasher should be unlocked after staged poison react level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsAssassinCrossUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobAssassinH},
		Skills: session.Skills{
			Points: 2,
			List: []session.Skill{
				{ID: db.SkillTFDouble, Level: 5, Upgradable: true},
				{ID: db.SkillTFPoison, Level: 10, Upgradable: true},
				{ID: db.SkillTFDetoxify, Level: 1, Upgradable: true},
				{ID: db.SkillASKatar, Level: 7, Upgradable: true},
				{ID: db.SkillASRight, Level: 3, Upgradable: true},
				{ID: db.SkillASCloaking, Level: 3, Upgradable: true},
				{ID: db.SkillASSonicblow, Level: 5, Upgradable: true},
				{ID: db.SkillASEnchantpoison, Level: 6, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillASCKatar,
		db.SkillASCCdp,
		db.SkillASCBreaker,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("assassin cross tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillASCEdp) {
		t.Fatal("enchant deadly poison should not be unlocked before create deadly poison reaches level 1")
	}
	if containsUnlockedSkill(skills, db.SkillASCMeteorassault) {
		t.Fatal("meteor assault should not be unlocked before soul destroyer reaches level 1")
	}
	window.stageSkill(db.SkillASCCdp)
	window.stageSkill(db.SkillASCBreaker)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillASCEdp) {
		t.Fatal("enchant deadly poison should be unlocked after staged create deadly poison satisfies robr prerequisite")
	}
	if !containsUnlockedSkill(skills, db.SkillASCMeteorassault) {
		t.Fatal("meteor assault should be unlocked after staged soul destroyer satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsRogueUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobRogue},
		Skills: session.Skills{
			Points: 2,
			List: []session.Skill{
				{ID: db.SkillTFSteal, Level: 1, Upgradable: true},
				{ID: db.SkillTFHiding, Level: 1, Upgradable: true},
				{ID: db.SkillACVulture, Level: 9, Upgradable: true},
				{ID: db.SkillRGSnatcher, Level: 4, Upgradable: true},
				{ID: db.SkillRGStealcoin, Level: 3, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillACVulture,
		db.SkillRGTunneldrive,
		db.SkillRGSnatcher,
		db.SkillRGStealcoin,
		db.SkillRGStriphelm,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("rogue tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillACDouble) {
		t.Fatal("double strafe should not be unlocked before vulture's eye reaches level 10")
	}
	if containsUnlockedSkill(skills, db.SkillRGBackstap) {
		t.Fatal("back stab should not be unlocked before mug reaches level 4")
	}
	window.stageSkill(db.SkillACVulture)
	window.stageSkill(db.SkillRGStealcoin)
	skills = window.visibleSkills(Context{Session: s})
	if !containsUnlockedSkill(skills, db.SkillACDouble) {
		t.Fatal("double strafe should be unlocked after staged vulture's eye level satisfies robr prerequisite")
	}
	if !containsUnlockedSkill(skills, db.SkillRGBackstap) {
		t.Fatal("back stab should be unlocked after staged mug level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsStalkerUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobRogueH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillTFHiding, Level: 5, Upgradable: true},
				{ID: db.SkillRGTunneldrive, Level: 3, Upgradable: true},
				{ID: db.SkillRGPlagiarism, Level: 10, Upgradable: true},
				{ID: db.SkillRGStripweapon, Level: 4, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillSTChasewalk,
		db.SkillSTPreserve,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("stalker tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillSTFullstrip) {
		t.Fatal("full divestment should not be unlocked before divest weapon reaches level 5")
	}
	window.stageSkill(db.SkillRGStripweapon)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillSTFullstrip) {
		t.Fatal("full divestment should be unlocked after staged divest weapon level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsBlacksmithUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobBlacksmith},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillBSIron, Level: 1, Upgradable: true},
				{ID: db.SkillBSEnchantedstone, Level: 1, Upgradable: true},
				{ID: db.SkillBSDagger, Level: 2, Upgradable: true},
				{ID: db.SkillBSSword, Level: 1, Upgradable: true},
				{ID: db.SkillBSHiltbinding, Level: 1, Upgradable: true},
				{ID: db.SkillBSSteel, Level: 1, Upgradable: true},
				{ID: db.SkillBSHammerfall, Level: 2, Upgradable: true},
				{ID: db.SkillBSWeaponresearch, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillBSSteel,
		db.SkillBSEnchantedstone,
		db.SkillBSOrideocon,
		db.SkillBSSword,
		db.SkillBSKnuckle,
		db.SkillBSSpear,
		db.SkillBSTwohandsword,
		db.SkillBSFindingore,
		db.SkillBSWeaponresearch,
		db.SkillBSRepairweapon,
		db.SkillBSAdrenaline,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("blacksmith tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillBSAxe) {
		t.Fatal("smith axe should not be unlocked before smith sword reaches level 2")
	}
	window.stageSkill(db.SkillBSSword)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillBSAxe) {
		t.Fatal("smith axe should be unlocked after staged smith sword level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsWhitesmithUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobBlacksmithH},
		Skills: session.Skills{
			Points: 3,
			List: []session.Skill{
				{ID: db.SkillMCPushcart, Level: 5, Upgradable: true},
				{ID: db.SkillMCCartrevolution, Level: 1, Upgradable: true},
				{ID: db.SkillMCChangecart, Level: 1, Upgradable: true},
				{ID: db.SkillMCMammonite, Level: 10, Upgradable: true},
				{ID: db.SkillBSHiltbinding, Level: 1, Upgradable: true},
				{ID: db.SkillBSSkintemper, Level: 3, Upgradable: true},
				{ID: db.SkillBSHammerfall, Level: 5, Upgradable: true},
				{ID: db.SkillBSWeaponresearch, Level: 9, Upgradable: true},
				{ID: db.SkillBSOverthrust, Level: 4, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillWSCartboost,
		db.SkillWSMeltdown,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("whitesmith tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	for _, skillID := range []uint16{
		db.SkillWSWeaponrefine,
		db.SkillWSOverthrustmax,
		db.SkillWSCarttermination,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("whitesmith skill %d should not be unlocked before staged prerequisite: %v", skillID, skills)
		}
	}
	window.stageSkill(db.SkillBSWeaponresearch)
	window.stageSkill(db.SkillBSOverthrust)
	window.stageSkill(db.SkillWSCartboost)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillWSWeaponrefine,
		db.SkillWSOverthrustmax,
		db.SkillWSCarttermination,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("whitesmith staged prerequisites did not expose skill %d: %v", skillID, skills)
		}
	}
}

func TestSkillWindowShowsAlchemistUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobAlchemist},
		Skills: session.Skills{
			Points: 4,
			List: []session.Skill{
				{ID: db.SkillAMLearningpotion, Level: 5, Upgradable: true},
				{ID: db.SkillAMPharmacy, Level: 9, Upgradable: true},
				{ID: db.SkillAMBioethics, Level: 1, Upgradable: true},
				{ID: db.SkillAMCpHelm, Level: 3, Upgradable: true},
				{ID: db.SkillAMCpShield, Level: 3, Upgradable: true},
				{ID: db.SkillAMCpArmor, Level: 2, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillAMPharmacy,
		db.SkillAMSpheremine,
		db.SkillAMPotionpitcher,
		db.SkillAMDemonstration,
		db.SkillAMAcidterror,
		db.SkillAMCannibalize,
		db.SkillAMCpHelm,
		db.SkillAMCpShield,
		db.SkillAMCpArmor,
		db.SkillAMRest,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("alchemist tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	for _, skillID := range []uint16{
		db.SkillAMTwilight1,
		db.SkillAMCpWeapon,
		db.SkillAMCallhomun,
	} {
		if containsUnlockedSkill(skills, skillID) {
			t.Fatalf("alchemist skill %d should not be unlocked before staged prerequisite: %v", skillID, skills)
		}
	}
	window.stageSkill(db.SkillAMPharmacy)
	window.stageSkill(db.SkillAMCpArmor)
	window.stageSkill(db.SkillAMRest)
	skills = window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillAMTwilight1,
		db.SkillAMTwilight2,
		db.SkillAMTwilight3,
		db.SkillAMCpWeapon,
		db.SkillAMCallhomun,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("alchemist staged prerequisites did not expose skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillAMResurrecthomun) {
		t.Fatal("homunculus resurrection should not be unlocked before call homunculus reaches level 1")
	}
	window.stageSkill(db.SkillAMCallhomun)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillAMResurrecthomun) {
		t.Fatal("homunculus resurrection should be unlocked after staged call homunculus level satisfies robr prerequisite")
	}
}

func TestSkillWindowShowsCreatorUnlocksFromRobrowserTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobAlchemistH},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillAMPotionpitcher, Level: 5, Upgradable: true},
				{ID: db.SkillAMCpWeapon, Level: 5, Upgradable: true},
				{ID: db.SkillAMCpArmor, Level: 5, Upgradable: true},
				{ID: db.SkillAMCpShield, Level: 5, Upgradable: true},
				{ID: db.SkillAMCpHelm, Level: 5, Upgradable: true},
				{ID: db.SkillAMDemonstration, Level: 5, Upgradable: true},
				{ID: db.SkillAMAcidterror, Level: 4, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}
	skills := window.visibleSkills(Context{Session: s})
	for _, skillID := range []uint16{
		db.SkillCRSlimpitcher,
		db.SkillCRFullprotection,
	} {
		if !containsUnlockedSkill(skills, skillID) {
			t.Fatalf("creator tree did not expose unlocked skill %d: %v", skillID, skills)
		}
	}
	if containsUnlockedSkill(skills, db.SkillCRAciddemonstration) {
		t.Fatal("acid demonstration should not be unlocked before acid terror reaches level 5")
	}
	window.stageSkill(db.SkillAMAcidterror)
	if !containsUnlockedSkill(window.visibleSkills(Context{Session: s}), db.SkillCRAciddemonstration) {
		t.Fatal("acid demonstration should be unlocked after staged acid terror level satisfies robr prerequisite")
	}
}

func TestSkillWindowOrdersPendingUnlocksBySkillTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSuperNovice},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMBash, Level: 1, Upgradable: true},
				{ID: db.SkillMGLightningbolt, Level: 3, Upgradable: true},
				{ID: db.SkillMGFirebolt, Level: 1, Upgradable: true},
				{ID: db.SkillALHeal, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}

	window.stageSkill(db.SkillMGLightningbolt)
	skills := window.visibleSkills(Context{Session: s})
	thunderstorm := skillIndex(skills, db.SkillMGThunderstorm)
	firebolt := skillIndex(skills, db.SkillMGFirebolt)
	heal := skillIndex(skills, db.SkillALHeal)
	if thunderstorm < 0 {
		t.Fatal("expected thunderstorm to be visible")
	}
	if !(firebolt < thunderstorm && thunderstorm < heal) {
		t.Fatalf("thunderstorm index = %d, firebolt = %d, heal = %d; expected skill-tree order", thunderstorm, firebolt, heal)
	}
}

func TestSkillWindowDoubleClickUsesSharedSkillController(t *testing.T) {
	ctx := Context{ScreenW: 800, ScreenH: 600}
	mode := &skillWindowTestRenderer{}
	window := &SkillWindow{selectedLevels: map[uint16]int{db.SkillSMProvoke: 1}}
	skill := session.Skill{ID: 6, Type: 1, Level: 2, Range: 9}

	window.pressSkill(ctx, mode, skill, 20, 30)
	if mode.used.ID != 0 {
		t.Fatalf("skill used after first click = %+v, want none", mode.used)
	}
	if window.dragSkill.Level != 1 {
		t.Fatalf("dragged skill level = %d, want selected level 1", window.dragSkill.Level)
	}

	window.pressSkill(ctx, mode, skill, 20, 30)
	if mode.used.ID != 6 || mode.used.Level != 1 {
		t.Fatalf("used skill = %+v, want provoke level 1", mode.used)
	}
}

func TestSkillWindowSelectedLevelClampsAndDoesNotRebuild(t *testing.T) {
	skill := session.Skill{ID: db.SkillMGSoulstrike, Type: 1, Level: 10}
	window := &SkillWindow{snapshot: "unchanged"}

	if got := window.selectedSkillLevel(skill); got != 10 {
		t.Fatalf("default selected level = %d, want learned level 10", got)
	}
	if !window.adjustSelectedSkillLevel(nil, skill, 0, -1) {
		t.Fatal("level decrement should change a selectable skill")
	}
	if got := window.selectedSkillLevel(skill); got != 9 {
		t.Fatalf("selected level after decrement = %d, want 9", got)
	}
	if window.dirty || window.snapshot != "unchanged" {
		t.Fatal("level selection should not trigger a skill-window rebuild")
	}
	for i := 0; i < 20; i++ {
		window.adjustSelectedSkillLevel(nil, skill, 0, -1)
	}
	if got := window.selectedSkillLevel(skill); got != 1 {
		t.Fatalf("selected level after repeated decrements = %d, want 1", got)
	}
	if window.adjustSelectedSkillLevel(nil, skill, 0, -1) {
		t.Fatal("decrement at level 1 should not report a change")
	}
	for i := 0; i < 20; i++ {
		window.adjustSelectedSkillLevel(nil, skill, 0, 1)
	}
	if got := window.selectedSkillLevel(skill); got != 10 {
		t.Fatalf("selected level after repeated increments = %d, want 10", got)
	}
	if _, ok := window.selectedLevels[skill.ID]; ok {
		t.Fatal("default learned level should not occupy the selection map")
	}
}

func TestSkillWindowFixedLevelSkillHasNoSelectionControls(t *testing.T) {
	skill := session.Skill{ID: db.SkillACDouble, Type: 1, Level: 10}
	window := &SkillWindow{}

	if window.adjustSelectedSkillLevel(nil, skill, 0, -1) {
		t.Fatal("fixed-level skill should reject level selection")
	}
	level := window.skillTableCell(Context{}, nil, skill, rotheme.TableViewCellContext{
		Column: rotheme.TableViewColumn{Key: "level"},
	})
	if level.Text != "10" {
		t.Fatalf("fixed-level cell text = %q, want 10", level.Text)
	}
	for _, key := range []string{"leveldown", "levelupselect"} {
		cell := window.skillTableCell(Context{}, nil, skill, rotheme.TableViewCellContext{
			Column: rotheme.TableViewColumn{Key: key},
		})
		if !cell.Hidden || cell.HasIconButton {
			t.Fatalf("fixed-level %s cell = %+v, want hidden", key, cell)
		}
	}
}

func TestSkillWindowSelectableLevelCellsShowCurrentAndMaximum(t *testing.T) {
	skill := session.Skill{ID: db.SkillMGSoulstrike, Type: 1, Level: 10}
	window := &SkillWindow{selectedLevels: map[uint16]int{skill.ID: 4}}

	level := window.skillTableCell(Context{}, nil, skill, rotheme.TableViewCellContext{
		Column: rotheme.TableViewColumn{Key: "level"},
	})
	if level.Text != "4/10" {
		t.Fatalf("selectable-level cell text = %q, want 4/10", level.Text)
	}
	if level.Align != widget.TextAlignCenter {
		t.Fatalf("selectable-level alignment = %v, want centered", level.Align)
	}
	down := window.skillTableCell(Context{}, nil, skill, rotheme.TableViewCellContext{
		Column: rotheme.TableViewColumn{Key: "leveldown"},
	})
	up := window.skillTableCell(Context{}, nil, skill, rotheme.TableViewCellContext{
		Column: rotheme.TableViewColumn{Key: "levelupselect"},
	})
	if !down.HasIconButton || down.IconButton != rotheme.IconButtonLeft || down.IconButtonDisabled {
		t.Fatalf("level-down cell = %+v, want enabled left arrow", down)
	}
	if !up.HasIconButton || up.IconButton != rotheme.IconButtonRight || up.IconButtonDisabled {
		t.Fatalf("level-up cell = %+v, want enabled right arrow", up)
	}
}

func TestSkillWindowSkillAtMouseUsesTableViewBody(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSwordman},
		Skills: session.Skills{
			List: []session.Skill{
				{ID: db.SkillSMRecovery},
				{ID: db.SkillSMBash},
			},
		},
	}
	window := &SkillWindow{}
	window.EnsureWindow(skillWindowWidth, skillWindowHeight)
	window.x = 20
	window.y = 30
	ctx := Context{Session: s}

	tableX := window.x + skillTabRailW + verticalTabDividerW
	if _, ok := window.skillAtMouse(ctx, tableX+8, window.y+ROWindowTitleHeight+skillHeaderH-1); ok {
		t.Fatal("header should not hit a skill row")
	}

	skill, ok := window.skillAtMouse(ctx, tableX+8, window.y+ROWindowTitleHeight+skillHeaderH+1)
	if !ok || skill.ID != db.SkillNVBasic {
		t.Fatalf("skill at top row = %+v, %v; want id %d", skill, ok, db.SkillNVBasic)
	}

	window.ensureScrollSignal().Set(skillRowH)
	skill, ok = window.skillAtMouse(ctx, tableX+8, window.y+ROWindowTitleHeight+skillHeaderH+1)
	if !ok || skill.ID != db.SkillNVFirstaid {
		t.Fatalf("skill at scrolled top row = %+v, %v; want id %d", skill, ok, db.SkillNVFirstaid)
	}
}

func TestSkillWindowGroupsSkillsIntoSharedVerticalTabs(t *testing.T) {
	const unknownSkillID uint16 = 65000
	s := &session.Session{
		Selected: session.Character{Job: db.JobKnightH},
		Skills: session.Skills{List: []session.Skill{
			{ID: db.SkillNVBasic, Level: 9},
			{ID: db.SkillSMBash, Level: 10},
			{ID: db.SkillKNPierce, Level: 5},
			{ID: db.SkillLKSpiralpierce, Level: 1},
			{ID: unknownSkillID, Level: 1},
		}},
	}
	window := &SkillWindow{}
	ctx := Context{Session: s}
	window.ensureSkillView(ctx)

	wantTabs := []int{skillTabFirst, skillTabSecond, skillTabEtc}
	if len(window.visibleTabs) != len(wantTabs) {
		t.Fatalf("visible tabs = %v, want %v", window.visibleTabs, wantTabs)
	}
	for i, want := range wantTabs {
		if window.visibleTabs[i] != want {
			t.Fatalf("visible tabs = %v, want %v", window.visibleTabs, wantTabs)
		}
	}
	if !containsSkill(window.activeSkills(), db.SkillNVBasic) || !containsSkill(window.activeSkills(), db.SkillSMBash) || containsSkill(window.activeSkills(), db.SkillKNPierce) {
		t.Fatalf("first tab skills = %+v, want novice and swordman skills only", window.activeSkills())
	}
	window.tab = skillTabSecond
	if !containsSkill(window.activeSkills(), db.SkillKNPierce) || !containsSkill(window.activeSkills(), db.SkillLKSpiralpierce) || containsSkill(window.activeSkills(), db.SkillSMBash) {
		t.Fatalf("second tab skills = %+v, want knight and lord knight skills only", window.activeSkills())
	}
	window.tab = skillTabEtc
	if !containsSkill(window.activeSkills(), unknownSkillID) {
		t.Fatalf("etc tab skills = %+v, want unknown server skill", window.activeSkills())
	}

	window.tab = skillTabFirst
	column := window.skillTabColumn(ctx, nil, nil)
	children := column.Children()
	if len(children) != len(wantTabs) {
		t.Fatalf("skill tab widgets = %d, want %d", len(children), len(wantTabs))
	}
	for i, child := range children {
		tab, ok := child.(*tabWidget)
		if !ok {
			t.Fatalf("skill tab child %d = %T, want *tabWidget", i, child)
		}
		if tab.cfg.labelRotation != rotheme.TextRotationCounterClockwise {
			t.Fatalf("skill tab %q rotation = %v, want counter-clockwise", tab.cfg.label, tab.cfg.labelRotation)
		}
	}
}

func TestSkillWindowCachesOrderedAndGroupedSkills(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSwordman},
		Skills:   session.Skills{List: []session.Skill{{ID: db.SkillSMBash, Level: 1}}},
	}
	window := &SkillWindow{}
	ctx := Context{Session: s}
	first := window.visibleSkills(ctx)
	second := window.visibleSkills(ctx)
	if len(first) == 0 || len(second) == 0 || &first[0] != &second[0] {
		t.Fatal("unchanged skill state should reuse the cached ordered view")
	}
}

func TestSkillDefaultPositionCentersOnScreen(t *testing.T) {
	x, y := skillWindowDefaultPosition(Context{ScreenW: 800, ScreenH: 600}, skillWindowWidth, skillWindowHeight)
	if x != 184 || y != 106 {
		t.Fatalf("skill default position = %d,%d; want centered 184,106", x, y)
	}

	x, y = skillWindowDefaultPosition(Context{ScreenW: 320, ScreenH: 240}, skillWindowWidth, skillWindowHeight)
	if x != windowScreenMargin || y != windowScreenMargin {
		t.Fatalf("small screen skill default position = %d,%d; want margin %d,%d", x, y, windowScreenMargin, windowScreenMargin)
	}
}

func TestSkillWindowTablePlusStagesSkill(t *testing.T) {
	skill := session.Skill{ID: db.SkillSMBash, Level: 1, MaxLevel: 10, Upgradable: true}
	s := &session.Session{Skills: session.Skills{Points: 1}}
	window := &SkillWindow{}
	bounds := skillTableButtonBounds(0, "levelup")

	consumed := window.handleSkillTableRowEvent(
		nil,
		Context{Session: s},
		nil,
		[]session.Skill{skill},
		0,
		event.NewMouseEvent(
			event.MousePress,
			event.ButtonLeft,
			event.ButtonStateLeft,
			geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
			geometry.Pt(100, 120),
			0,
		),
	)

	if !consumed {
		t.Fatal("plus press was not consumed")
	}
	if got := window.pendingFor(skill.ID); got != 1 {
		t.Fatalf("pending skill levels = %d, want 1", got)
	}
	if !window.dirty {
		t.Fatal("plus press should mark the window dirty")
	}
}

func TestSkillWindowMonkListOnlyUpgradesAvailableSkills(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobMonk},
		Skills: session.Skills{Points: 3, List: []session.Skill{
			{ID: db.SkillALDp, Level: 10},
			{ID: db.SkillALDemonbane, Level: 9, Upgradable: true},
		}},
	}
	ctx := Context{Session: s}
	window := &SkillWindow{tab: skillTabSecond}
	for _, unlocked := range []bool{false, true} {
		if unlocked {
			window.stageSkill(db.SkillALDemonbane)
		}
		window.ensureSkillView(ctx)
		skills := window.activeSkills()
		row := skillIndex(skills, db.SkillMOIronhand)
		if row < 0 {
			t.Fatal("Iron Fists is missing from the Monk list")
		}
		cell := window.skillTableCell(ctx, nil, skills[row], rotheme.TableViewCellContext{
			Column: rotheme.TableViewColumn{Key: "levelup"},
		})
		if cell.HasIconButton != unlocked {
			t.Fatalf("level-up button visible=%t, want %t", cell.HasIconButton, unlocked)
		}
		if !unlocked {
			typeCell := window.skillTableCell(ctx, nil, skills[row], rotheme.TableViewCellContext{
				Column: rotheme.TableViewColumn{Key: "type"},
			})
			if !typeCell.Hidden {
				t.Fatal("locked skill without server type information should not be labeled passive")
			}
		}
		point := skillTableButtonBounds(row, "levelup").Center()
		window.handleSkillTableRowEvent(nil, ctx, nil, skills, row,
			event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, 0))
		if got := window.pendingFor(db.SkillMOIronhand); (got == 1) != unlocked {
			t.Fatalf("pending Iron Fists = %d, unlocked=%t", got, unlocked)
		}
	}
}

func TestSkillWindowTableLevelArrowSelectsWithoutStartingDrag(t *testing.T) {
	skill := session.Skill{ID: db.SkillMGSoulstrike, Type: 1, Level: 10}
	window := &SkillWindow{}
	bounds := skillTableButtonBounds(0, "leveldown")

	consumed := window.handleSkillTableRowEvent(
		nil,
		Context{},
		nil,
		[]session.Skill{skill},
		0,
		event.NewMouseEvent(
			event.MousePress,
			event.ButtonLeft,
			event.ButtonStateLeft,
			geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
			geometry.Pt(100, 120),
			0,
		),
	)

	if !consumed {
		t.Fatal("level arrow press was not consumed")
	}
	if got := window.selectedSkillLevel(skill); got != 9 {
		t.Fatalf("selected level = %d, want 9", got)
	}
	if window.dragActive {
		t.Fatal("level arrow press should not start a skill drag")
	}
	if window.dirty {
		t.Fatal("level arrow press should not rebuild the skill window")
	}
}

func TestSkillWindowBlankLevelArrowColumnStillStartsFixedSkillDrag(t *testing.T) {
	skill := session.Skill{ID: db.SkillACDouble, Type: 1, Level: 10}
	window := &SkillWindow{}
	bounds := skillTableButtonBounds(0, "leveldown")

	window.handleSkillTableRowEvent(
		nil,
		Context{},
		nil,
		[]session.Skill{skill},
		0,
		event.NewMouseEvent(
			event.MousePress,
			event.ButtonLeft,
			event.ButtonStateLeft,
			geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
			geometry.Pt(100, 120),
			0,
		),
	)

	if !window.dragActive || window.dragSkill.ID != skill.ID {
		t.Fatalf("fixed-level drag = active %t skill %+v, want skill %d", window.dragActive, window.dragSkill, skill.ID)
	}
}

func TestSkillDragReleaseOverShortcutStoresSkill(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

	window := &SkillWindow{
		dragSkill:  session.Skill{ID: 46, Level: 10, Type: 1},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	if !window.UpdateDrag(Context{Input: inputState, ScreenW: 800, ScreenH: 600}, bar) {
		t.Fatal("skill drag release was not consumed")
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != 46 || got.skillLevel != 10 {
		t.Fatalf("shortcut slot = %+v, want double strafe level 10", got)
	}
}

func TestSkillDragReleaseStoresLevelSelectedWithArrows(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

	skill := session.Skill{ID: db.SkillMGSoulstrike, Level: 10, Type: 1}
	window := &SkillWindow{selectedLevels: map[uint16]int{skill.ID: 4}}
	window.pressSkill(Context{}, nil, skill, 20, 30)
	if !window.UpdateDrag(Context{Input: inputState, ScreenW: 800, ScreenH: 600}, bar) {
		t.Fatal("selected-level skill drag release was not consumed")
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != skill.ID || got.skillLevel != 4 {
		t.Fatalf("shortcut slot = %+v, want soul strike level 4", got)
	}
}

func TestSkillDragReleaseOverShortcutRejectsPassiveSkill(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	bar.slots[0] = shortcutSlotState{kind: shortcutItem, itemID: 501}
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

	window := &SkillWindow{
		dragSkill:  session.Skill{ID: db.SkillALDemonbane, Level: 10, Type: 0},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	if !window.UpdateDrag(Context{Input: inputState, ScreenW: 800, ScreenH: 600}, bar) {
		t.Fatal("passive skill drag release over shortcut should be consumed")
	}
	if got := bar.slots[0]; got.kind != shortcutItem || got.itemID != 501 {
		t.Fatalf("shortcut slot = %+v, want unchanged item shortcut", got)
	}
}

func TestSkillWindowRejectsPassiveDrag(t *testing.T) {
	window := &SkillWindow{}
	window.pressSkill(
		Context{},
		nil,
		session.Skill{ID: db.SkillALDemonbane, Type: 0, Level: 10},
		20,
		30,
	)
	if window.dragActive {
		t.Fatal("passive player skill started a drag")
	}
}

func TestSkillTooltipUsesSharedOverlayState(t *testing.T) {
	window := &SkillWindow{}
	ctx := Context{ScreenW: 800, ScreenH: 600}
	skill := session.Skill{ID: 6, Name: "Provoke", Level: 2, Range: 9}

	window.showTooltip(ctx, skill, 100, 120)
	if !window.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	text := window.tooltip.Text()
	if !strings.Contains(text, "Provoke") || !strings.Contains(text, "Lv 2") || !strings.Contains(text, "Range: 9") {
		t.Fatalf("tooltip text = %q", text)
	}

	window.hideTooltip()
	if window.tooltip.Open() {
		t.Fatal("tooltip should be closed")
	}
}

func (w *SkillWindow) visibleSkills(ctx Context) []session.Skill {
	w.ensureSkillView(ctx)
	return w.allSkills
}

func TestSkillTooltipPreservesDescriptionColors(t *testing.T) {
	desc := "Type:^33cc00 Active^000000\nCatalyst:^ee0000 1 Red Gemstone^000000\nEffect:^777777 A long description with enough words to wrap onto another line.^000000"
	manager, err := res.NewManagerFS(fstest.MapFS{
		"data/skilldesctable.txt": {Data: []byte("MG_STONECURSE#\nStone Curse\n" + desc + "\n#\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	window := &SkillWindow{}
	window.showTooltip(Context{Resources: manager}, session.Skill{ID: db.SkillMGStonecurse}, 100, 120)
	if !window.tooltip.colorCodes || !window.tooltip.Open() {
		t.Fatal("skill tooltip did not enable text colors")
	}
	want := strings.Join(append([]string{"Stone Curse", "Lv 0", ""}, wrapItemInfoLines(strings.Split(desc, "\n"), 38)...), "\n")
	if got := window.tooltip.Text(); got != want {
		t.Fatalf("tooltip text = %q, want %q", got, want)
	}
	colored := wrapItemInfoLines(strings.Split(desc, "\n"), 38)
	plain := wrapItemInfoLines(strings.Split(rotheme.StripColorCodes(desc), "\n"), 38)
	if rotheme.StripColorCodes(strings.Join(colored, "\n")) != strings.Join(plain, "\n") {
		t.Fatal("description color codes changed the text wrapping")
	}
	window.tooltip.Show(Context{}, "Plain tooltip", 100, 120, 80)
	if window.tooltip.colorCodes {
		t.Fatal("plain tooltip retained the previous skill's color mode")
	}
}

type skillWindowTestRenderer struct {
	used session.Skill
}

func (r *skillWindowTestRenderer) DrawInventoryItemIcon(*render.Frame, *res.Manager, session.InventoryItem, int, int) {
}

func (r *skillWindowTestRenderer) DrawSkillIcon(*render.Frame, *res.Manager, session.Skill, int, int, int) {
}

func (r *skillWindowTestRenderer) SkillIconImage(*res.Manager, session.Skill, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) ItemInfoIllustrationImage(*res.Manager, session.InventoryItem, int, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) EquipmentPreviewImage(Context, int, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) EquipmentPreviewImageForCharacter(Context, session.Character, byte, int, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) UseShortcutSkill(_ Context, skill session.Skill) error {
	r.used = skill
	return nil
}

func containsUnlockedSkill(skills []session.Skill, skillID uint16) bool {
	index := skillIndex(skills, skillID)
	return index >= 0 && (skills[index].Level > 0 || skills[index].Upgradable)
}

func containsSkill(skills []session.Skill, skillID uint16) bool {
	return skillIndex(skills, skillID) >= 0
}

func skillIndex(skills []session.Skill, skillID uint16) int {
	for i, skill := range skills {
		if skill.ID == skillID {
			return i
		}
	}
	return -1
}
