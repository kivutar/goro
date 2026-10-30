package ui

import (
	"testing"

	"github.com/kivutar/goro/network"
)

func TestTeleportModalSelectsFirstEnabledDestination(t *testing.T) {
	modal := TeleportModal{}

	modal.OpenWarpPointList(network.WarpPointList{
		SkillID:  teleportSkillID,
		MapNames: []string{teleportRandomMap},
	})

	if modal.row != 0 {
		t.Fatalf("selected row = %d, want Random", modal.row)
	}
	destinations := modal.destinations()
	if len(destinations) != 2 {
		t.Fatalf("destination count = %d, want 2", len(destinations))
	}
	if !destinations[0].enabled || destinations[1].enabled {
		t.Fatalf("destination enabled states = %#v, want Random enabled and Save Point disabled", destinations)
	}
}

func TestTeleportModalWarpPortalDestinationsSkipEmptySlots(t *testing.T) {
	modal := TeleportModal{}

	modal.OpenWarpPointList(network.WarpPointList{
		SkillID:  warpPortalSkillID,
		MapNames: []string{"prontera", "", "geffen"},
	})

	destinations := modal.destinations()
	if len(destinations) != 2 {
		t.Fatalf("destination count = %d, want 2", len(destinations))
	}
	if destinations[0].label != "Save Point: prontera" || destinations[0].mapName != "prontera" {
		t.Fatalf("first destination = %#v", destinations[0])
	}
	if destinations[1].label != "geffen" || destinations[1].mapName != "geffen" {
		t.Fatalf("second destination = %#v", destinations[1])
	}
	if modal.row != 0 {
		t.Fatalf("selected row = %d, want first warp destination", modal.row)
	}
}

func TestTeleportModalSavePointFollowsDestinations(t *testing.T) {
	for _, tc := range []struct {
		name    string
		maps    []string
		enabled bool
	}{
		{"empty", nil, false},
		{"random only", []string{"Random"}, false},
		{"save point", []string{"Random", "prontera"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var modal TeleportModal
			modal.OpenWarpPointList(network.WarpPointList{SkillID: teleportSkillID, MapNames: tc.maps})
			destination := modal.destinations()[1]
			if destination.enabled != tc.enabled {
				t.Fatalf("save point enabled=%t, want %t", destination.enabled, tc.enabled)
			}
			if tc.enabled && destination.mapName != "SavePoint" {
				t.Fatalf("save point selection=%q, want SavePoint", destination.mapName)
			}
		})
	}
}
