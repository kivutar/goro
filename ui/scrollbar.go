package ui

const ROScrollbarGutter = 12

func scrollbarSafeIntWidth(width int) int {
	if width <= ROScrollbarGutter {
		return 0
	}
	return width - ROScrollbarGutter
}
