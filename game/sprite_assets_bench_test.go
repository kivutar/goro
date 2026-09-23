package game

import (
	"os"
	"testing"

	"github.com/kivutar/goro/res"
)

// Measures preparation of new views, including after map view caches reset.
// Run on the same data directory before/after changes for a useful comparison.
func BenchmarkRepeatedHumanoidAppearances(b *testing.B) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		b.Skip("set GORO_DATA_DIR to benchmark real client sprites")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		for _, archive := range manager.Archives {
			_ = archive.Close()
		}
	})
	load := func(head int) {
		if view, status := loadHumanoidSpriteView(manager, 4, head, 1, 0, 0, "acolyte"); view == nil {
			b.Fatal(status)
		}
	}
	for head := 1; head <= 3; head++ {
		load(head)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		load(1 + i%3)
	}
}
