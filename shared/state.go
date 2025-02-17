package shared

// State contains common game state that both game and network use.
type State struct {
	BallX      float32
	BallY      float32
	BallVX     float32
	BallVY     float32
	P1X        float32
	P1Y        float32
	P2X        float32
	P2Y        float32
	ScoreLeft  int
	ScoreRight int
	Timestamp  int64
}

// InterpolateState linearly interpolates between two States by t.
func InterpolateState(s1, s2 State, t float32) State {
	return State{
		BallX:      s1.BallX + (s2.BallX-s1.BallX)*t,
		BallY:      s1.BallY + (s2.BallY-s1.BallY)*t,
		BallVX:     s1.BallVX + (s2.BallVX-s1.BallVX)*t,
		BallVY:     s1.BallVY + (s2.BallVY-s1.BallVY)*t,
		P1X:        s1.P1X + (s2.P1X-s1.P1X)*t,
		P1Y:        s1.P1Y + (s2.P1Y-s1.P1Y)*t,
		P2X:        s1.P2X + (s2.P2X-s1.P2X)*t,
		P2Y:        s1.P2Y + (s2.P2Y-s1.P2Y)*t,
		ScoreLeft:  s2.ScoreLeft, // Use s2 directly (or choose differently)
		ScoreRight: s2.ScoreRight,
		Timestamp:  s1.Timestamp + int64(float32(s2.Timestamp-s1.Timestamp)*t),
	}
}
