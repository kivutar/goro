package battle

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

func TestHallucinationFollowsLocalStatus(t *testing.T) {
	start := time.Unix(100, 0)
	for _, id := range []uint32{0, 2000000, 150000, 300} {
		s := &session.Session{AccountID: 2000000, CharID: 150000}
		frame := render.NewFrame(800, 600)
		active := func(at time.Time) bool {
			frame.BeginFrame()
			ApplyHallucination(frame, s, at)
			return reflect.ValueOf(frame).Elem().FieldByName("hallucination").Bool()
		}
		if active(start) {
			t.Fatal("hallucination active without status")
		}
		change := network.StatusEffectChange{StatusID: db.StatusIllusion, ActorID: id, Active: true, HasDuration: true, Duration: time.Second}
		ApplyLocalStatusEffectChange(s, change, start)
		if got := active(start); got != (id != 300) {
			t.Fatalf("actor %d: active=%v", id, got)
		}
		s.NoEffects = true
		if active(start) {
			t.Fatal("/effect did not hide hallucination")
		}
		s.NoEffects = false
		if got := active(start.Add(time.Millisecond)); got != (id != 300) {
			t.Fatal("/effect discarded the underlying status")
		}
		if active(start.Add(time.Second)) {
			t.Fatal("hallucination outlived its status")
		}
		change.HasDuration = false
		ApplyLocalStatusEffectChange(s, change, start)
		if got := active(start.Add(time.Hour)); got != (id != 300) {
			t.Fatal("indefinite status expired")
		}
		change.Active = false
		ApplyLocalStatusEffectChange(s, change, start)
		if active(start) {
			t.Fatal("hallucination survived status removal")
		}
	}
}
