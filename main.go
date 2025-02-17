package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"

	"pong-multiplayer/engine"
	"pong-multiplayer/game"
	"pong-multiplayer/network"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	buttonWidth  = 200
	buttonHeight = 50
)

// MenuState defines which part of the menu is active.
type MenuState int

const (
	MenuMain MenuState = iota
	MenuJoinInput
)

const defaultRenderDelay = int64(100 * 1e6) // 100ms in ns

func main() {
	eng, err := engine.NewEngine("Multiplayer Pong", 800, 600)
	if err != nil {
		log.Fatalf("Engine initialization failed: %v", err)
	}
	defer eng.Shutdown()

	// Initialize TTF
	if err := ttf.Init(); err != nil {
		log.Fatalf("TTF initialization failed: %v", err)
	}
	defer ttf.Quit()

	// Open a font (ensure the TTF file exists in your working directory)
	font, err := ttf.OpenFont("arial.ttf", 16)
	if err != nil {
		log.Fatalf("Failed to open font: %v", err)
	}
	defer font.Close()

	var state MenuState = MenuMain
	var selectedMode string   // "host" or "join"
	var joinInviteCode string // entered by the joining player

	// Define button rectangles.
	hostBtn := sdl.Rect{X: 300, Y: 200, W: buttonWidth, H: buttonHeight}
	joinBtn := sdl.Rect{X: 300, Y: 300, W: buttonWidth, H: buttonHeight}

	// When hosting, generate an invite code.
	inviteCode := generateInviteCode()

	// Declare a buffer for received states.
	var stateBuffer []game.State

	// Main menu loop.
	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch ev := event.(type) {
			case *sdl.QuitEvent:
				return
			case *sdl.MouseButtonEvent:
				if ev.Type == sdl.MOUSEBUTTONDOWN {
					x, y := ev.X, ev.Y
					if state == MenuMain {
						if pointInRect(int32(x), int32(y), hostBtn) {
							selectedMode = "host"
						} else if pointInRect(int32(x), int32(y), joinBtn) {
							selectedMode = "join"
							state = MenuJoinInput
							joinInviteCode = ""
							sdl.StartTextInput()
						}
					}
				}
			case *sdl.TextInputEvent:
				if state == MenuJoinInput {
					// Convert the fixed C array to a Go slice and trim at the first null byte.
					textArr := (*[32]byte)(unsafe.Pointer(&ev.Text))
					n := 0
					for ; n < len(textArr) && textArr[n] != 0; n++ {
					}
					joinInviteCode += string(textArr[:n])
				}
			case *sdl.KeyboardEvent:
				if state == MenuJoinInput && ev.Type == sdl.KEYDOWN {
					if ev.Keysym.Sym == sdl.K_BACKSPACE && len(joinInviteCode) > 0 {
						joinInviteCode = joinInviteCode[:len(joinInviteCode)-1]
					} else if ev.Keysym.Sym == sdl.K_RETURN {
						// Finished entering invite code.
						state = MenuMain
						sdl.StopTextInput()
						selectedMode = "join"
					}
				}
			}
		}

		// Render the menu.
		eng.Renderer.SetDrawColor(50, 50, 50, 255)
		eng.Renderer.Clear()

		if state == MenuMain {
			drawButton(eng.Renderer, hostBtn, "Host ("+inviteCode+")")
			drawButton(eng.Renderer, joinBtn, "Join")
		} else if state == MenuJoinInput {
			// Draw input area.
			inputRect := sdl.Rect{X: 300, Y: 400, W: buttonWidth, H: buttonHeight}
			eng.Renderer.SetDrawColor(200, 200, 200, 255)
			eng.Renderer.FillRect(&inputRect)
			eng.Renderer.SetDrawColor(0, 0, 0, 255)
			eng.Renderer.DrawRect(&inputRect)

			// Render the typed text.
			if joinInviteCode != "" {
				surface, err := font.RenderUTF8Blended(joinInviteCode, sdl.Color{R: 0, G: 0, B: 0, A: 255})
				if err == nil {
					texture, err := eng.Renderer.CreateTextureFromSurface(surface)
					if err == nil {
						var tw, th int32
						_, _, tw, th, err = texture.Query()
						if err == nil {
							dst := sdl.Rect{X: inputRect.X + 5, Y: inputRect.Y + (inputRect.H-th)/2, W: tw, H: th}
							eng.Renderer.Copy(texture, nil, &dst)
						}
						texture.Destroy()
					}
					surface.Free()
				}
			}
		}

		eng.Renderer.Present()
		sdl.Delay(16)
		// Exit menu loop if a mode is selected.
		if selectedMode != "" && state == MenuMain {
			break
		}
	}

	// In host mode, start the server in a goroutine and log the invite code.
	if selectedMode == "host" {
		// Create the server with the expected invite code.
		server := network.NewServer("localhost:9000", inviteCode)
		go func() {
			if err := server.Start(); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		}()
		log.Printf("Hosting game. Invite code: %s", inviteCode)

		// Immediately connect as client using the generated invite code.
		client := network.NewClient("localhost:9000")
		if err := client.Connect(inviteCode); err != nil {
			log.Fatalf("Client connection failed: %v", err)
		}

		// Wait until at least one remote client has connected.
		// Because the host's own connection is in the Clients map,
		// we assume a remote player has joined when Clients has >= 2.
		for {
			server.Lock.Lock()
			count := len(server.Clients)
			server.Lock.Unlock()
			if count >= 2 {
				break
			}
			sdl.Delay(16)
		}

		// Create the game instance.
		g := game.NewGame(eng)

		// Initialize a sequence counter.
		var seq uint32 = 0

		// Set a callback on the server so that when an "input_update" message is received,
		// the RemoteInput field is updated (expecting msg.Data to be an integer as a string).
		server.InputUpdate = func(msg network.Message) {
			direction, err := strconv.Atoi(string(msg.Data))
			if err != nil {
				direction = 0
			}
			g.RemoteInput = direction
		}

		// Broadcast state updates to all connected clients.
		go func() {
			for g.Engine.Running {
				seq++
				state := g.GetState()
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, state.BallX)
				binary.Write(buf, binary.BigEndian, state.BallY)
				binary.Write(buf, binary.BigEndian, state.BallVX)
				binary.Write(buf, binary.BigEndian, state.BallVY)
				binary.Write(buf, binary.BigEndian, state.P1X)
				binary.Write(buf, binary.BigEndian, state.P1Y)
				binary.Write(buf, binary.BigEndian, state.P2X)
				binary.Write(buf, binary.BigEndian, state.P2Y)
				binary.Write(buf, binary.BigEndian, int32(state.ScoreLeft))
				binary.Write(buf, binary.BigEndian, int32(state.ScoreRight))
				binary.Write(buf, binary.BigEndian, state.Timestamp)

				msg := network.Message{
					Type: network.MessageTypeStateUpdate,
					Seq:  seq,
					Data: buf.Bytes(),
				}
				server.Broadcast(msg)
				sdl.Delay(10)
			}
		}()

		g.Run()
	} else if selectedMode == "join" {
		log.Printf("Joining game with invite code: %s", joinInviteCode)
		client := network.NewClient("localhost:9000")
		if err := client.Connect(joinInviteCode); err != nil {
			log.Printf("Failed to join the game: %v", err)
			return
		}

		// Create the game instance. Its state will be updated via server broadcasts.
		g := game.NewGame(eng)

		// Create a buffered channel for input updates.
		inputChan := make(chan int, 20)

		// Start a goroutine dedicated to sending input updates without blocking.
		go func() {
			for eng.Running {
				select {
				case direction := <-inputChan:
					msg := network.Message{
						Type: network.MessageTypeInputUpdate,
						Data: []byte(fmt.Sprintf("%d", direction)),
					}
					if err := client.Send(msg); err != nil {
						fmt.Println("Error sending input update:", err)
					}
				default:
					sdl.Delay(1)
				}
			}
		}()

		// Use a state-update callback to reconcile the host's state with local prediction.
		client.OnStateUpdate = func(s game.State) {
			// Append the newly received state.
			stateBuffer = append(stateBuffer, s)

			// Remove very old states (older than 300ms, for example).
			now := time.Now().UnixNano()
			delay := int64(300 * 1e6) // 300ms in nanoseconds
			for len(stateBuffer) >= 2 && stateBuffer[0].Timestamp < now-delay {
				stateBuffer = stateBuffer[1:]
			}
		}

		// Run a combined render and input loop.
		lastRender := time.Now()

		for eng.Running {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch ev := event.(type) {
				case *sdl.QuitEvent:
					eng.Running = false
				case *sdl.KeyboardEvent:
					// Capture UP and DOWN events.
					if ev.Keysym.Sym == sdl.K_UP || ev.Keysym.Sym == sdl.K_DOWN {
						var direction int
						if ev.Type == sdl.KEYDOWN {
							if ev.Keysym.Sym == sdl.K_UP {
								direction = -1
							} else if ev.Keysym.Sym == sdl.K_DOWN {
								direction = 1
							}
						} else if ev.Type == sdl.KEYUP {
							direction = 0
						}
						// Immediately update the local paddle (client-side prediction).
						g.Player2.Y += float32(direction) * g.Player2.Speed * (10.0 / 1000.0)
						if g.Player2.Y < 0 {
							g.Player2.Y = 0
						} else if g.Player2.Y+float32(g.Player2.Height) > 600 {
							g.Player2.Y = 600 - float32(g.Player2.Height)
						}
						// Non-blocking send to the input channel.
						select {
						case inputChan <- direction:
						default:
							// If the channel is full, drop the input update.
						}
					}
				}
			}
			// Use an adaptive render delay based on measuredRTT (set by your ping/pong routine).
			adaptiveDelay := defaultRenderDelay
			if network.MeasuredRTT > 0 {
				// A more precise value is obtained by reading atomically.
				adaptiveDelay = atomic.LoadInt64(&network.MeasuredRTT) / 2
			}

			if len(stateBuffer) >= 2 {
				renderTime := time.Now().UnixNano() - adaptiveDelay

				// Find the two states surrounding renderTime.
				var s1, s2 game.State
				for i := 0; i < len(stateBuffer)-1; i++ {
					if stateBuffer[i].Timestamp <= renderTime && renderTime <= stateBuffer[i+1].Timestamp {
						s1 = stateBuffer[i]
						s2 = stateBuffer[i+1]
						break
					}
				}
				duration := s2.Timestamp - s1.Timestamp
				if duration > 0 {
					t := float32(renderTime-s1.Timestamp) / float32(duration)
					interpolatedState := game.InterpolateState(s1, s2, t)
					// For join clients, update remote objects only.
					g.ApplyRemoteState(interpolatedState, true)
				}
			}

			// Calculate FPS and retrieve the RTT atomically.
			now := time.Now()
			dt := now.Sub(lastRender)
			fps := 1.0 / dt.Seconds()
			lastRender = now

			// Clear, render game, then display overlay text.
			eng.Clear()
			g.Render()
			pingMs := int64(0)
			rtt := atomic.LoadInt64(&network.MeasuredRTT)
			if rtt > 0 {
				pingMs = rtt / 1e6
			}
			infoText := fmt.Sprintf("FPS: %.0f  Ping: %d ms", fps, pingMs)
			if err := renderText(eng.Renderer, font, infoText, 10, 10); err != nil {
				fmt.Println("Error rendering info text:", err)
			}
			eng.Present()
			sdl.Delay(16)
		}
	}
}

func pointInRect(x, y int32, r sdl.Rect) bool {
	return x >= r.X && x <= (r.X+r.W) && y >= r.Y && y <= (r.Y+r.H)
}

func drawButton(renderer *sdl.Renderer, rect sdl.Rect, label string) {
	// Draw a simple colored button.
	renderer.SetDrawColor(100, 100, 255, 255)
	renderer.FillRect(&rect)
	// (Optionally, render the label using a text library such as SDL_ttf.)
}

func generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#@"
	code := make([]byte, 6)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func renderText(renderer *sdl.Renderer, font *ttf.Font, text string, x, y int32) error {
	color := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	surface, err := font.RenderUTF8Solid(text, color)
	if err != nil {
		return err
	}
	defer surface.Free()
	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return err
	}
	defer texture.Destroy()
	rect := sdl.Rect{X: x, Y: y, W: surface.W, H: surface.H}
	return renderer.Copy(texture, nil, &rect)
}
