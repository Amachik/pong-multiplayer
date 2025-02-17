package game

import (
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
)

// Player represents a paddle in the game.
type Player struct {
	X, Y           float32
	Width, Height  int32
	Speed          float32
	UpKey, DownKey sdl.Scancode
}

// NewPlayer creates a new player (paddle) at the specified position.
func NewPlayer(x, y float32) *Player {
	return &Player{
		X:      x,
		Y:      y,
		Width:  10,
		Height: 100,
		Speed:  300, // pixels per second
		// Default keys (can be overridden later)
		UpKey:   sdl.Scancode(sdl.SCANCODE_UP),
		DownKey: sdl.Scancode(sdl.SCANCODE_DOWN),
	}
}

// Update handles paddle movement based on keyboard input.
func (p *Player) Update(deltaTime float32) {
	keys := sdl.GetKeyboardState()
	if keys[p.UpKey] != 0 {
		p.Y -= p.Speed * deltaTime
	}
	if keys[p.DownKey] != 0 {
		p.Y += p.Speed * deltaTime
	}
	// Clamp within window bounds (assuming window height 600)
	if p.Y < 0 {
		p.Y = 0
	} else if p.Y+float32(p.Height) > 600 {
		p.Y = 600 - float32(p.Height)
	}
}

// Render draws the paddle with rounded borders.
func (p *Player) Render(renderer *sdl.Renderer) {
	// Convert paddle position and dimensions to int32.
	x1 := int32(p.X)
	y1 := int32(p.Y)
	x2 := x1 + p.Width
	y2 := y1 + p.Height
	// Use a smaller radius so that the interior gets filled.
	radius := int32(2)

	// Draw a filled rounded rectangle.
	if ok := gfx.RoundedBoxRGBA(renderer, x1, y1, x2, y2, radius, 255, 255, 255, 255); !ok {
		// If drawing the rounded rectangle fails, fall back to a standard rectangle.
		renderer.SetDrawColor(255, 255, 255, 255)
		rect := sdl.Rect{X: x1, Y: y1, W: p.Width, H: p.Height}
		renderer.FillRect(&rect)
	}
}
