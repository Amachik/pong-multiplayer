package engine

import "github.com/veandco/go-sdl2/sdl"

// ProcessInput polls SDL events and returns false if a quit event is received.
func ProcessInput() bool {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			return false
		}
	}
	return true
}
