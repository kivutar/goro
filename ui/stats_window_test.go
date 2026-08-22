package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestStatsWindowSectionsLayOutHorizontally(t *testing.T) {
	w := &StatsWindow{}
	body := w.statsBodyWidget(Context{Session: &session.Session{}})
	body.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(
		float32(statsWindowWidth-2*statsWindowPad),
		statsWindowHeight-ROWindowTitleHeight-2*statsWindowPad,
	)))
	children := body.(interface{ Children() []widget.Widget }).Children()
	if len(children) != 2 {
		t.Fatalf("stats body children = %d, want primary and derived sections", len(children))
	}
	primaryChildren := children[0].(interface{ Children() []widget.Widget }).Children()
	if len(primaryChildren) != 1 {
		t.Fatalf("primary stats children = %d, want only stat rows", len(primaryChildren))
	}
	primary := children[0].(interface{ Bounds() geometry.Rect }).Bounds()
	right := children[1].(interface{ Bounds() geometry.Rect }).Bounds()
	if right.Min.X < primary.Max.X {
		t.Fatalf("stats sections overlap horizontally: primary=%v right=%v", primary, right)
	}
	if right.Min.Y != primary.Min.Y {
		t.Fatalf("stats sections top alignment differs: primary=%v right=%v", primary, right)
	}
	bodyBounds := body.(interface{ Bounds() geometry.Rect }).Bounds()
	if right.Max.X != bodyBounds.Max.X {
		t.Fatalf("stats body has trailing horizontal space: body=%v right=%v", bodyBounds, right)
	}
	rightChildren := children[1].(interface{ Children() []widget.Widget }).Children()
	if len(rightChildren) != 2 {
		t.Fatalf("right stats children = %d, want derived and detail tables", len(rightChildren))
	}
	derived := rightChildren[0].(interface{ Bounds() geometry.Rect }).Bounds()
	details := rightChildren[1].(interface{ Bounds() geometry.Rect }).Bounds()
	if details.Min.Y != derived.Max.Y+rotheme.TableGap {
		t.Fatalf("stats detail table y = %.1f, want %.1f", details.Min.Y, derived.Max.Y+rotheme.TableGap)
	}
	if details.Width() != derived.Width() {
		t.Fatalf("stats table widths differ: derived=%.1f details=%.1f", derived.Width(), details.Width())
	}
}

func TestStatsGuildNameUsesAvailableSessionState(t *testing.T) {
	tests := []struct {
		name string
		s    *session.Session
		want string
	}{
		{name: "no session", want: ""},
		{name: "no guild", s: &session.Session{}, want: ""},
		{name: "actor name", s: &session.Session{GuildName: " Knights "}, want: "Knights"},
		{name: "guild details", s: &session.Session{GuildName: "Old", Guild: session.Guild{Name: "Mandala"}}, want: "Mandala"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := statsGuildName(test.s); got != test.want {
				t.Fatalf("stats guild name = %q, want %q", got, test.want)
			}
		})
	}
}

func TestStatsDetailsSplitAtDerivedTableCenter(t *testing.T) {
	details := statsDetailsWidget(&session.Session{GuildName: "Knights"}, 12)
	details.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(
		statsDerivedColumnWidth,
		2*statsRowH+rotheme.TableGap,
	)))
	if got := details.(interface{ Bounds() geometry.Rect }).Bounds().Width(); got != statsDerivedColumnWidth {
		t.Fatalf("stats details width = %.1f, want %.1f", got, statsDerivedColumnWidth)
	}
	if got := 2*statsDerivedPairWidth + rotheme.TableGap; got != statsDerivedColumnWidth {
		t.Fatalf("stats detail columns total = %.1f, want %.1f", got, statsDerivedColumnWidth)
	}
}

func TestStatsSnapshotChangesWithGuildName(t *testing.T) {
	s := &session.Session{}
	before := statsWindowSnapshot(s)
	s.GuildName = "Knights"
	if after := statsWindowSnapshot(s); after == before {
		t.Fatal("stats snapshot did not change with guild name")
	}
}

func TestStatsPrimaryRowsUseTableRhythm(t *testing.T) {
	w := &StatsWindow{}
	rows := w.statRowsWidget(Context{Session: &session.Session{}})
	rows.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(
		statsPrimaryColumnWidth,
		6*statsRowH+5*statsRowGap,
	)))
	children := rows.(interface{ Children() []widget.Widget }).Children()
	if len(children) != 6 {
		t.Fatalf("primary stat rows = %d, want 6", len(children))
	}
	for i, child := range children {
		bounds := child.(interface{ Bounds() geometry.Rect }).Bounds()
		if bounds.Height() != statsRowH {
			t.Fatalf("primary row %d height = %.1f, want %d", i, bounds.Height(), statsRowH)
		}
		wantY := float32(i) * (statsRowH + statsRowGap)
		if bounds.Min.Y != wantY {
			t.Fatalf("primary row %d y = %.1f, want %.1f", i, bounds.Min.Y, wantY)
		}
		rowChildren := child.Children()
		if len(rowChildren) == 0 || len(rowChildren[0].Children()) != 1 {
			t.Fatalf("primary row %d label cell is missing", i)
		}
		label, ok := rowChildren[0].Children()[0].(*primitives.TextWidget)
		if !ok {
			t.Fatalf("primary row %d label = %T, want text", i, rowChildren[0].Children()[0])
		}
		style := label.Style()
		if style.Color != rotheme.Default.Colors.LabelText || !style.Bold || style.FontFamily != "" {
			t.Fatalf("primary row %d label does not use semantic label style", i)
		}
	}
}

func TestStatPointCostPreRenewal(t *testing.T) {
	tests := []struct {
		current int
		want    int
	}{
		{current: 1, want: 2},
		{current: 10, want: 2},
		{current: 11, want: 3},
		{current: 20, want: 3},
		{current: 91, want: 11},
	}
	for _, test := range tests {
		if got := statPointCost(test.current); got != test.want {
			t.Fatalf("statPointCost(%d) = %d, want %d", test.current, got, test.want)
		}
	}
}

func TestStatCostPrefersServerValue(t *testing.T) {
	row := statRow{value: 31, cost: 8}
	if got := statCost(row); got != 8 {
		t.Fatalf("statCost() = %d, want 8", got)
	}
}
