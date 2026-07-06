package ui

import "testing"

func TestCharacterSelectPage(t *testing.T) {
	if got := CharacterSelectPage(5); got != 1 {
		t.Fatalf("page = %d, want 1", got)
	}
}

func TestCharacterCreateGraphDrawOrderIsValidHexagon(t *testing.T) {
	points := CharacterCreateGraphPoints(0, 0, 64)
	order := CharacterCreateGraphDrawOrder()
	seen := map[int]bool{}
	for _, stat := range order {
		if stat < 0 || stat >= CharacterCreateStatCount {
			t.Fatalf("stat index outside range in graph order: %d", stat)
		}
		if seen[stat] {
			t.Fatalf("duplicate stat index in graph order: %d", stat)
		}
		seen[stat] = true
	}

	if points[CharacterCreateStatDex][0] >= 0 || points[CharacterCreateStatDex][1] <= 0 {
		t.Fatalf("DEX graph point = %#v, want lower-left", points[CharacterCreateStatDex])
	}
	if points[CharacterCreateStatLuk][0] <= 0 || points[CharacterCreateStatLuk][1] <= 0 {
		t.Fatalf("LUK graph point = %#v, want lower-right", points[CharacterCreateStatLuk])
	}

	for i := 0; i < CharacterCreateStatCount; i++ {
		a1 := points[order[i]]
		a2 := points[order[(i+1)%CharacterCreateStatCount]]
		for j := i + 1; j < CharacterCreateStatCount; j++ {
			if graphEdgesAdjacent(i, j) {
				continue
			}
			b1 := points[order[j]]
			b2 := points[order[(j+1)%CharacterCreateStatCount]]
			if graphSegmentsCross(a1, a2, b1, b2) {
				t.Fatalf("graph edges %d and %d cross", i, j)
			}
		}
	}
}

func graphEdgesAdjacent(a, b int) bool {
	if a == b {
		return true
	}
	if a+1 == b || b+1 == a {
		return true
	}
	return (a == 0 && b == CharacterCreateStatCount-1) || (b == 0 && a == CharacterCreateStatCount-1)
}

func graphSegmentsCross(a1, a2, b1, b2 [2]float64) bool {
	o1 := graphOrientation(a1, a2, b1)
	o2 := graphOrientation(a1, a2, b2)
	o3 := graphOrientation(b1, b2, a1)
	o4 := graphOrientation(b1, b2, a2)
	return o1*o2 < 0 && o3*o4 < 0
}

func graphOrientation(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}
