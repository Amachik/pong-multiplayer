package game

import (
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
)

type Ball struct {
	X, Y   float32
	VX, VY float32
	Size   int32 // represents the diameter of the ball
}

func NewBall(x, y float32) *Ball {
	return &Ball{
		X:    x,
		Y:    y,
		VX:   250, // pixels per second
		VY:   250,
		Size: 20, // Increased size for a smoother, rounder ball.
	}
}

func (b *Ball) Update(deltaTime float32) {
	b.X += b.VX * deltaTime
	b.Y += b.VY * deltaTime

	// Bounce off top and bottom walls (assuming window height 600)
	if b.Y <= 0 {
		b.Y = 0
		b.VY = -b.VY
	} else if b.Y+float32(b.Size) >= 600 {
		b.Y = 600 - float32(b.Size)
		b.VY = -b.VY
	}
}

func (b *Ball) Render(renderer *sdl.Renderer) {
	// Calculate the center of the ball and its radius directly as int32 values.
	centerX := int32(b.X + float32(b.Size)/2)
	centerY := int32(b.Y + float32(b.Size)/2)
	radius := b.Size / 2

	// Draw a filled circle for the ball.
	if ok := gfx.FilledCircleRGBA(renderer, centerX, centerY, radius, 255, 255, 255, 255); !ok {
		// If drawing the circle fails, fall back to drawing a rectangle.
		renderer.SetDrawColor(255, 255, 255, 255)
		rect := sdl.Rect{X: int32(b.X), Y: int32(b.Y), W: b.Size, H: b.Size}
		renderer.FillRect(&rect)
	}
}
