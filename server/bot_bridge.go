package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"server/internal/engine"
	"strings"
	"time"
)

const (
	sherlockNickname  = "Sherlock"
	pythonActionURL   = "http://ai-agent:5000/act"
	pythonActionTimeo = 2 * time.Second
)

var pythonActionClient = &http.Client{Timeout: pythonActionTimeo}

type botActionRequest struct {
	CurrentTurnPlayer string                     `json:"current_turn_player"`
	Players           []engine.PlayerObservation `json:"players"`
	Bullets           []engine.BulletObservation `json:"bullets"`
}

type botActionResponse struct {
	Action string `json:"action"`
}

func (r *Room) spawnSherlock() {
	if r == nil || r.Engine == nil {
		return
	}

	if err := r.Engine.AddPlayer(sherlockNickname); err != nil {
		return
	}
	r.Engine.SetPlayerBot(sherlockNickname, true)
}

func (r *Room) syncBotAction() {
	if r == nil || r.Engine == nil {
		return
	}

	state := r.Engine.GameState()
	if !hasBotPlayer(state) {
		return
	}

	currentTurnPlayer := sherlockNickname
	for _, player := range state.Players {
		if player.IsBot {
			currentTurnPlayer = player.Name
			break
		}
	}

	action := requestBotAction(botActionRequest{
		CurrentTurnPlayer: currentTurnPlayer,
		Players:           state.Players,
		Bullets:           state.Bullets,
	})
	applyBotAction(r.Engine, currentTurnPlayer, action)
}

func (r *Room) clearBotInput() {
	if r == nil || r.Engine == nil {
		return
	}
	r.Engine.SetPlayerInput(sherlockNickname, engine.PlayerInput{})
}

func hasBotPlayer(state engine.Observation) bool {
	for _, player := range state.Players {
		if player.IsBot {
			return true
		}
	}
	return false
}

func requestBotAction(payload botActionRequest) string {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "STOP"
	}

	req, err := http.NewRequest(http.MethodPost, pythonActionURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "STOP"
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pythonActionClient.Do(req)
	if err != nil {
		return "STOP"
	}
	defer resp.Body.Close()

	var result botActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "STOP"
	}

	action := strings.ToUpper(strings.TrimSpace(result.Action))
	switch action {
	case "UP", "DOWN", "LEFT", "RIGHT", "FIRE", "STOP":
		return action
	default:
		return "STOP"
	}
}

func applyBotAction(e engine.Engine, nickname string, action string) {
	input := engine.PlayerInput{}

	switch action {
	case "UP":
		input.Up = true
		e.SetPlayerInput(nickname, input)
	case "DOWN":
		input.Down = true
		e.SetPlayerInput(nickname, input)
	case "LEFT":
		input.Left = true
		e.SetPlayerInput(nickname, input)
	case "RIGHT":
		input.Right = true
		e.SetPlayerInput(nickname, input)
	case "FIRE":
		e.SetPlayerInput(nickname, input)
		e.TryFire(nickname)
	default:
		e.SetPlayerInput(nickname, input)
	}
}
