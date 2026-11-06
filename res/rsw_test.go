package res

import (
	"bytes"
	"testing"
)

func TestParseRSW(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("GRSW")
	buf.WriteByte(2)
	buf.WriteByte(7)
	writeI32(&buf, 186)
	buf.WriteByte(0)
	writeFixedRSWString(&buf, "test.ini", 40)
	writeFixedRSWString(&buf, "test.gnd", 40)
	writeFixedRSWString(&buf, "test.gat", 40)
	writeFixedRSWString(&buf, "test.rsw", 40)

	writeI32(&buf, 90)
	writeI32(&buf, 35)
	writeF32(&buf, 0.8)
	writeF32(&buf, 0.7)
	writeF32(&buf, 0.6)
	writeF32(&buf, 0.2)
	writeF32(&buf, 0.3)
	writeF32(&buf, 0.4)
	writeF32(&buf, 0.5)

	writeI32(&buf, -100)
	writeI32(&buf, 100)
	writeI32(&buf, -200)
	writeI32(&buf, 200)

	writeI32(&buf, 2)
	writeI32(&buf, 11)
	writeI32(&buf, 22)

	writeI32(&buf, 4)

	writeI32(&buf, 1)
	writeFixedRSWString(&buf, "model-a", 40)
	writeI32(&buf, 2)
	writeF32(&buf, 1.5)
	writeI32(&buf, 3)
	buf.WriteByte(9)
	writeI32(&buf, 12)
	writeFixedRSWString(&buf, "building.rsm", 80)
	writeFixedRSWString(&buf, "main", 80)
	writeVector3(&buf, 10, 20, 30)
	writeVector3(&buf, 1, 2, 3)
	writeVector3(&buf, 5, 10, 15)

	writeI32(&buf, 2)
	writeFixedRSWString(&buf, "light-a", 80)
	writeVector3(&buf, 15, 25, 35)
	writeI32(&buf, 300)
	writeI32(&buf, 64)
	writeI32(&buf, -1)
	writeF32(&buf, 12.5)

	writeI32(&buf, 3)
	writeFixedRSWString(&buf, "sound-a", 80)
	writeFixedRSWString(&buf, "amb.wav", 80)
	writeVector3(&buf, 5, 10, 15)
	writeF32(&buf, 0.8)
	writeI32(&buf, 20)
	writeI32(&buf, 30)
	writeF32(&buf, 40)
	writeF32(&buf, 3.5)

	writeI32(&buf, 4)
	writeFixedRSWString(&buf, "effect-a", 80)
	writeVector3(&buf, 50, 60, 70)
	writeI32(&buf, 77)
	writeF32(&buf, 2.5)
	for _, value := range []float32{1, 2, 3, 4} {
		writeF32(&buf, value)
	}

	rsw, err := ParseRSW(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if rsw.VersionMajor != 2 || rsw.VersionMinor != 7 || rsw.BuildNumber != 186 {
		t.Fatalf("unexpected header: %+v", rsw)
	}
	if rsw.Files.GND != "test.gnd" || rsw.Files.GAT != "test.gat" || rsw.Files.SRC != "test.rsw" {
		t.Fatalf("unexpected files: %+v", rsw.Files)
	}
	if rsw.Light.Longitude != 90 || rsw.Light.Latitude != 35 || rsw.Light.Opacity != 0.5 {
		t.Fatalf("unexpected light: %+v", rsw.Light)
	}
	if rsw.Ground.Left != -200 || rsw.Ground.Right != 200 {
		t.Fatalf("unexpected ground: %+v", rsw.Ground)
	}
	if len(rsw.Models) != 1 || rsw.Models[0].Filename != "building.rsm" || rsw.Models[0].Position.X != 2 || rsw.Models[0].Position.Y != -4 || rsw.Models[0].Scale.Z != 3 {
		t.Fatalf("unexpected models: %+v", rsw.Models)
	}
	if len(rsw.Lights) != 1 || rsw.Lights[0].Position.Y != -5 || rsw.Lights[0].Color.R != 255 || rsw.Lights[0].Color.B != 0 {
		t.Fatalf("unexpected lights: %+v", rsw.Lights)
	}
	if len(rsw.Sounds) != 1 || rsw.Sounds[0].File != "amb.wav" || rsw.Sounds[0].Position.Y != -2 || rsw.Sounds[0].Cycle != 3.5 {
		t.Fatalf("unexpected sounds: %+v", rsw.Sounds)
	}
	if len(rsw.Effects) != 1 || rsw.Effects[0].Position.Y != -12 || rsw.Effects[0].Delay != 2.5 || rsw.Effects[0].Param[3] != 4 {
		t.Fatalf("unexpected effects: %+v", rsw.Effects)
	}
}

func writeFixedRSWString(buf *bytes.Buffer, value string, size int) {
	buf.WriteString(value)
	buf.Write(make([]byte, size-len(value)))
}

func writeVector3(buf *bytes.Buffer, x, y, z float32) {
	writeF32(buf, x)
	writeF32(buf, y)
	writeF32(buf, z)
}
