package engine

import (
	"github.com/veandco/go-sdl2/sdl"
)

type Engine struct {
	Window   *sdl.Window
	Renderer *sdl.Renderer
	Running  bool
}

func NewEngine(title string, width, height int32) (*Engine, error) {
	if err := sdl.Init(0x00000001); err != nil {
		return nil, err
	}

	sdl.SetHint("SDL_RENDER_SCALE_QUALITY", "linear")

	window, err := sdl.CreateWindow(title,
		int32(sdl.WINDOWPOS_CENTERED),
		int32(sdl.WINDOWPOS_CENTERED),
		width, height, uint32(sdl.WINDOW_SHOWN))
	if err != nil {
		return nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, uint32(sdl.RENDERER_ACCELERATED))
	if err != nil {
		return nil, err
	}

	return &Engine{
		Window:   window,
		Renderer: renderer,
		Running:  true,
	}, nil
}

func (e *Engine) Shutdown() {
	e.Renderer.Destroy()
	e.Window.Destroy()
	sdl.Quit()
}

func (e *Engine) Clear() {
	e.Renderer.SetDrawColor(0, 0, 0, 255) // Black background
	e.Renderer.Clear()
}

func (e *Engine) Present() {
	e.Renderer.Present()
}
