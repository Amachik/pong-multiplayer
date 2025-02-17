package engine

import "github.com/veandco/go-sdl2/sdl"

// DrawRect draws a filled rectangle with the specified color.
func DrawRect(renderer *sdl.Renderer, x, y, w, h int32, r, g, b, a uint8) error {
	renderer.SetDrawColor(r, g, b, a)
	rect := sdl.Rect{X: x, Y: y, W: w, H: h}
	return renderer.FillRect(&rect)
}
