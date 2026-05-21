package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/internal/engine"
)

const pythonServerURL = "http://localhost:5000/act"

type TrainingPayload struct {
	CurrentTurnPlayer string                     `json:"current_turn_player"`
	IsGameOver        bool                       `json:"isGameOver"`
	Players           []engine.PlayerObservation `json:"players"`
	Bullets           []engine.BulletObservation `json:"bullets"`
}

func startTrainingBridge() {
	fmt.Println("preparing the simulation ..")
	gameSim := engine.NewGameEngine()

	gameSim.AddPlayer("bot1")
	gameSim.AddPlayer("bot2")
	gameSim.AddPlayer("bot3")
	gameSim.AddPlayer("bot4")

	fmt.Println("training is starting ..")

	for episode := 0; episode < 10000; episode++ {
		gameSim.ResetRound()

		for len(gameSim.AlivePlayerNames()) > 1 {
			currentMapState := gameSim.GameState()
			isOver := len(gameSim.AlivePlayerNames()) <= 1

			for _, botName := range gameSim.AlivePlayerNames() {
				payload := TrainingPayload{
					CurrentTurnPlayer: botName,
					IsGameOver:        isOver,
					Players:           currentMapState.Players,
					Bullets:           currentMapState.Bullets,
				}

				actionStr := sendStateToPython(payload)
				applyActionToEngine(gameSim, botName, actionStr)
			}

			gameSim.Update(0.016)
		}

		if episode%10 == 0 {
			fmt.Printf("[BRIDGE] Round (Episode) %d completed.\n", episode)
		}
	}
}

func applyActionToEngine(gameSim engine.Engine, nickname string, action string) {
	input := engine.PlayerInput{Up: false, Down: false, Left: false, Right: false}

	switch action {
	case "UP":
		input.Up = true
		gameSim.SetPlayerInput(nickname, input)
	case "DOWN":
		input.Down = true
		gameSim.SetPlayerInput(nickname, input)
	case "LEFT":
		input.Left = true
		gameSim.SetPlayerInput(nickname, input)
	case "RIGHT":
		input.Right = true
		gameSim.SetPlayerInput(nickname, input)
	case "FIRE":
		gameSim.TryFire(nickname)
	}
}

func sendStateToPython(payload TrainingPayload) string {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("json transfer error: %v", err)
		return "STOP"
	}

	resp, err := http.Post(pythonServerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("training is stopped: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Printf("response reading error: %v", err)
		return "STOP"
	}
	return result["action"]
}
