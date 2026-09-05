package network

import (
	"bytes"
	"testing"
)

func TestBuildAutoRevivePacket(t *testing.T) {
	if got, want := BuildAutoRevivePacket(), []byte{0x92, 0x02}; !bytes.Equal(got, want) {
		t.Fatalf("auto-revive packet = % X, want % X", got, want)
	}
}
