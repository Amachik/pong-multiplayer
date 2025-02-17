package game

import (
	"fmt"
	"sync/atomic"
	"time"

	"pong-multiplayer/engine"
	"pong-multiplayer/network"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	windowWidth  = 800
	windowHeight = 600
)

// Game represents the game instance.
type Game struct {
	Engine      *engine.Engine
	Ball        *Ball
	Player1     *Player
	Player2     *Player
	ScoreLeft   int
	ScoreRight  int
	RemoteInput int // -1 for up, +1 for down, 0 for no input (controls Player2)
}

// State represents the minimal game state to share with clients.
type State struct {
	BallX, BallY          float32
	BallVX, BallVY        float32
	P1X, P1Y              float32
	P2X, P2Y              float32
	ScoreLeft, ScoreRight int
	Timestamp             int64
}

func NewGame(e *engine.Engine) *Game {
	// Create players at their starting positions.
	// Host (Player1) is on the left and uses W/S;
	// Remote player (Player2) is on the right.
	p1 := NewPlayer(30, 250)  // Left paddle
	p2 := NewPlayer(760, 250) // Right paddle

	// Set Player1 controls to W/S.
	p1.UpKey = sdl.Scancode(sdl.SCANCODE_W)
	p1.DownKey = sdl.Scancode(sdl.SCANCODE_S)

	return &Game{
		Engine:      e,
		Ball:        NewBall(float32(windowWidth/2-5), float32(windowHeight/2-5)),
		Player1:     p1,
		Player2:     p2,
		ScoreLeft:   0,
		ScoreRight:  0,
		RemoteInput: 0,
	}
}

func (g *Game) Update(deltaTime float32) {
	g.Ball.Update(deltaTime)
	g.Player1.Update(deltaTime)
	// Instead of using local keyboard state for Player2,
	// update with input provided by the remote client.
	g.updateRemotePlayer(deltaTime)

	g.checkPaddleCollision(g.Player1)
	g.checkPaddleCollision(g.Player2)

	// Example score logic – adjust as needed.
	if g.Ball.X < 0 {
		// Ball left the screen: right player scores.
		g.ScoreRight++
		g.resetBall()
	} else if g.Ball.X > windowWidth {
		// Ball went off right side: left player scores.
		g.ScoreLeft++
		g.resetBall()
	}

	// (Optionally, only the host can update the window title)
	title := fmt.Sprintf("Multiplayer Pong - Left: %d | Right: %d", g.ScoreLeft, g.ScoreRight)
	g.Engine.Window.SetTitle(title)
}

func (g *Game) updateRemotePlayer(deltaTime float32) {
	// Update Player2's vertical position using RemoteInput.
	g.Player2.Y += float32(g.RemoteInput) * g.Player2.Speed * deltaTime
	// Clamp within window height (assumed 600).
	if g.Player2.Y < 0 {
		g.Player2.Y = 0
	} else if g.Player2.Y+float32(g.Player2.Height) > windowHeight {
		g.Player2.Y = windowHeight - float32(g.Player2.Height)
	}
}

func (g *Game) checkPaddleCollision(p *Player) {
	// Simple AABB collision detection.
	if g.Ball.X <= p.X+float32(p.Width) &&
		g.Ball.X+float32(g.Ball.Size) >= p.X &&
		g.Ball.Y+float32(g.Ball.Size) >= p.Y &&
		g.Ball.Y <= p.Y+float32(p.Height) {
		// Reverse the ball's horizontal direction.
		g.Ball.VX = -g.Ball.VX
	}
}

func (g *Game) resetBall() {
	// Place the ball back in the center.
	g.Ball.X = float32(windowWidth)/2 - float32(g.Ball.Size)/2
	g.Ball.Y = float32(windowHeight)/2 - float32(g.Ball.Size)/2

	// Reset velocity (for simplicity, use fixed speeds).
	if g.Ball.VX < 0 {
		g.Ball.VX = 250
	} else {
		g.Ball.VX = -250
	}
	g.Ball.VY = 250
}

func (g *Game) Render() {
	g.Ball.Render(g.Engine.Renderer)
	g.Player1.Render(g.Engine.Renderer)
	g.Player2.Render(g.Engine.Renderer)
}

func (g *Game) Run() {
	var lastTime uint64 = sdl.GetTicks64()
	for g.Engine.Running {
		// Process input (host's local controls will update Player1 via Player.Update)
		g.Engine.Running = engine.ProcessInput()

		currentTime := sdl.GetTicks64()
		deltaTime := float32(currentTime-lastTime) / 1000.0
		lastTime = currentTime

		g.Update(deltaTime)
		g.Engine.Clear()
		g.Render()
		g.Engine.Present()
		sdl.Delay(16)
	}
}

// GetState returns the current game state.
func (g *Game) GetState() State {
	return State{
		BallX:      g.Ball.X,
		BallY:      g.Ball.Y,
		BallVX:     g.Ball.VX,
		BallVY:     g.Ball.VY,
		P1X:        g.Player1.X,
		P1Y:        g.Player1.Y,
		P2X:        g.Player2.X,
		P2Y:        g.Player2.Y,
		ScoreLeft:  g.ScoreLeft,
		ScoreRight: g.ScoreRight,
		Timestamp:  time.Now().UnixNano(),
	}
}

// SetState applies a given state to the game instance.
func (g *Game) SetState(s State) {
	g.Ball.X = s.BallX
	g.Ball.Y = s.BallY
	g.Ball.VX = s.BallVX
	g.Ball.VY = s.BallVY
	g.Player1.X = s.P1X
	g.Player1.Y = s.P1Y
	g.Player2.X = s.P2X
	g.Player2.Y = s.P2Y
	g.ScoreLeft = s.ScoreLeft
	g.ScoreRight = s.ScoreRight
}

// SetStateSmooth applies a received state smoothly to the game instance.
func (g *Game) SetStateSmooth(s State) {
	// Update ball and local player (Player1) immediately.
	g.Ball.X = s.BallX
	g.Ball.Y = s.BallY
	g.Ball.VX = s.BallVX
	g.Ball.VY = s.BallVY
	g.Player1.X = s.P1X
	g.Player1.Y = s.P1Y

	// Smoothly update Player2 (client's paddle)
	const smoothing = 0.2
	g.Player2.X += smoothing * (s.P2X - g.Player2.X)
	g.Player2.Y += smoothing * (s.P2Y - g.Player2.Y)

	// Directly update the score so that it stays in sync.
	g.ScoreLeft = s.ScoreLeft
	g.ScoreRight = s.ScoreRight
}

// ApplyRemoteState updates only the remote objects (and score)
// without altering the local player's paddle.
func (g *Game) ApplyRemoteState(s State, isClient bool) {
	// Always update the ball and score.
	g.Ball.X = s.BallX
	g.Ball.Y = s.BallY
	g.Ball.VX = s.BallVX
	g.Ball.VY = s.BallVY
	g.ScoreLeft = s.ScoreLeft
	g.ScoreRight = s.ScoreRight

	if isClient {
		// On the join client, update the host paddle (Player1) only,
		// leaving the client-controlled paddle (Player2) as updated by local input.
		g.Player1.X = s.P1X
		g.Player1.Y = s.P1Y
	} else {
		// On the host, update the remote paddle (Player2) smoothly.
		const smoothing = 0.2
		g.Player2.X += smoothing * (s.P2X - g.Player2.X)
		g.Player2.Y += smoothing * (s.P2Y - g.Player2.Y)
	}
}

func InterpolateState(s1, s2 State, t float32) State {
	return State{
		BallX:      s1.BallX + t*(s2.BallX-s1.BallX),
		BallY:      s1.BallY + t*(s2.BallY-s1.BallY),
		BallVX:     s1.BallVX + t*(s2.BallVX-s1.BallVX),
		BallVY:     s1.BallVY + t*(s2.BallVY-s1.BallVY),
		P1X:        s1.P1X + t*(s2.P1X-s1.P1X),
		P1Y:        s1.P1Y + t*(s2.P1Y-s1.P1Y),
		P2X:        s1.P2X + t*(s2.P2X-s1.P2X),
		P2Y:        s1.P2Y + t*(s2.P2Y-s1.P2Y),
		ScoreLeft:  s2.ScoreLeft, // Use the latest score
		ScoreRight: s2.ScoreRight,
		Timestamp:  s2.Timestamp,
	}
}

func (g *Game) RunOverlay(font *ttf.Font) {
	lastRender := time.Now()
	for g.Engine.Running {
		dt := time.Since(lastRender)
		lastRender = time.Now()
		g.Update(float32(dt.Seconds()))
		g.Engine.Clear()
		g.Render()
		rtt := atomic.LoadInt64(&network.MeasuredRTT)
		pingMs := int64(0)
		if rtt > 0 {
			pingMs = rtt / 1e6
		}
		infoText := fmt.Sprintf("FPS: %.0f  Ping: %d ms", 1.0/dt.Seconds(), pingMs)
		if err := renderText(g.Engine.Renderer, font, infoText, 10, 10); err != nil {
			fmt.Println("Error rendering info text:", err)
		}
		g.Engine.Present()
		sdl.Delay(16)
	}
}
