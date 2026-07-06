package game

import (
	"github.com/kivutar/goro/render"
)

func clear(screen *render.Image) {
	screen.Fill(render.ColorBackground)
}
