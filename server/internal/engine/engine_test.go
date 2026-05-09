package engine

import (
	"math"
	"slices"
	"testing"
	"time"
)

func TestIsWall(t *testing.T) {
	tests := []struct {
		name     string
		x, y     float64
		expected bool
	}{
		{"negative x", -1, 60, true},
		{"negative y", 60, -1, true},
		{"too large x", 900, 60, true},
		{"too large y", 60, 900, true},
		{"open area ", 60, 60, false},
		{"wall tile (4,2)", 160, 80, true},
		{"wall tile (6,3)", 240, 120, true},
		{"outer wall top", 40, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWall(tt.x, tt.y)
			if result != tt.expected {
				t.Errorf("isWall(%.0f,%.0f)=%v,expected %v",
					tt.x, tt.y, result, tt.expected)

			}
		})

	}
}

// add player testing
func TestAddPlayer(t *testing.T) {
	t.Run("succesfull addition", func(t *testing.T) { //succesfull addition testing
		engine := NewGameEngine()
		err := engine.AddPlayer("sherlock")
		if err != nil {
			t.Errorf("error was not expected %v", err)
		}
		if _, ok := engine.Players["sherlock"]; !ok {
			t.Errorf("sherlock could not be added.")

		}

	})

	t.Run("repeated nickname", func(t *testing.T) { //repeated nickname testing
		engine := NewGameEngine()
		engine.AddPlayer("ugur")
		err := engine.AddPlayer("ugur")
		if err != ErrNicknameTaken {
			t.Errorf("Repeated nickname error not detected.")
		}

	})

	t.Run("full room", func(t *testing.T) { //full room testing
		engine := NewGameEngine()
		engine.AddPlayer("player1")
		engine.AddPlayer("player2")
		engine.AddPlayer("player3")
		engine.AddPlayer("player4")
		err := engine.AddPlayer("player5")

		if err != ErrRoomFull {
			t.Errorf("ErrRoomFull not detected an error")
		}

	})

}

// reset_round testing
func TestResetRound(t *testing.T) {

	t.Run("bullet cleared", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("ugur")
		engine.AddBullet("ugur", 30, 60, 32)
		engine.AddPlayer("sherlock")
		engine.AddBullet("sherlock", 42, 64, 21)
		engine.ResetRound()
		if len(engine.Bullets) != 0 {
			t.Errorf("The bullets were not cleaned.")
		}

	})
	t.Run("health reset check", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.Players["sherlock"].Health = 0
		engine.ResetRound()
		if engine.Players["sherlock"].Health != 100 || engine.Players["sherlock"].X != 60 || engine.Players["sherlock"].Y != 60 {
			t.Errorf("player reset not work")
		}

	})

}

// alive player list testing
func TestAlivePlayerNames(t *testing.T) {
	engine := NewGameEngine()
	engine.AddPlayer("sherlock")
	engine.AddPlayer("ugur")
	engine.Players["sherlock"].Health = 0
	names := engine.AlivePlayerNames()
	if slices.Contains(names, "sherlock") {
		t.Errorf("sherlock shouldn't be on the list")

	}
	if !slices.Contains(names, "ugur") {
		t.Errorf("ugur should be on the list ")

	}
}

//tryfire function testing

func TestTryFire(t *testing.T) {
	t.Run("can fire control", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.Players["sherlock"].Health = 0
		engine.TryFire("sherlock")
		if len(engine.Bullets) != 0 {
			t.Errorf("fire control not working")
		}

	})
	t.Run("250 ms check", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.Players["sherlock"].LastShotAt = time.Now()
		engine.TryFire("sherlock")
		if len(engine.Bullets) != 0 {
			t.Errorf("250 ms error")
		}

	})
	t.Run("check normal fire", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.TryFire("sherlock")
		if len(engine.Bullets) != 1 {
			t.Errorf("try_fire did not to at list")
		}
	})

}

func TestSimulationStep(t *testing.T) {
	engine := NewGameEngine()
	engine.AddPlayer("bot")
	engine.ResetRound()

	initialX := engine.Players["bot"].X

	// 100 step, action 0 (up/forward)
	for i := 0; i < 100; i++ {
		engine.Step(0)
	}

	finalX := engine.Players["bot"].X

	if finalX == initialX {
		t.Errorf("bot should have moved after 100 steps")
	}

	if finalX <= initialX {
		t.Errorf("bot should move forward, but X decreased or stayed same")
	}
}


func TestUpdate(t *testing.T) {
	dt := 0.01 // 10ms
	t.Run("dead player move", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.Players["sherlock"].Input.Up = true
		engine.Players["sherlock"].Health = 0
		engine.Update(dt)
		if engine.Players["sherlock"].X != 60 || engine.Players["sherlock"].Y != 60 {
			t.Errorf("dead player should not move")
		}
	})

	t.Run("living movement player check", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.Players["sherlock"].Input.Up = true
		engine.Update(dt)
		if engine.Players["sherlock"].X == 60 {
			t.Errorf("living player must be able to move")
		}

	})

	t.Run("wall obstacle control", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.Players["sherlock"].X = TankRadius + 1
		engine.Players["sherlock"].Angle = math.Pi
		engine.Players["sherlock"].Input.Up = true
		engine.Update(dt)
		if engine.Players["sherlock"].X != TankRadius+1 {
			t.Errorf("wall collision not working")
		}
	})

	t.Run("bullet TTL check ", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("sherlock")
		engine.AddBullet("sherlock", 100, 100, 0)
		engine.Bullets[0].CreatedAt = time.Now().Add(-3 * time.Second)
		engine.Update(dt)
		if len(engine.Bullets) != 0 {
			t.Errorf("expired bullets should be removed from the slice")

		}
	})

	t.Run("bullet hit deteciton", func(t *testing.T) {
		engine := NewGameEngine()
		engine.AddPlayer("target")
		engine.AddPlayer("shooter")
		engine.AddBullet("shooter", 60, 60, 0)
		engine.Update(dt)
		if engine.Players["target"].Health > 0 {
			t.Errorf("target player should have 0 health after being hit")
		}
	})

}
