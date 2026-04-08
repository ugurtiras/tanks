package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const MaxPlayersPerRoom = 4

type Room struct {
	ID          string
	Owner       string
	GameStarted bool
	GameOver    bool
	Winner      string
	Clients     map[*Client]bool
	Broadcast   chan []byte //odaya özel metin iletim kanalı
	Register    chan *Client
	Unregister  chan *Client
	Closed      chan string
	Engine      *GameEngine
}

func NewRoom(id string, closed chan string) *Room {
	return &Room{
		ID:         id,
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Closed:     closed,
		Engine:     NewGameEngine(), //motoru oluştur
	}
}

func (r *Room) Run() {
	fmt.Printf("Oda %s started\n", r.ID)

	//oyun döngüsü icin ticker (16ms)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		//oyuncu katıldığında
		case client := <-r.Register:
			r.Clients[client] = true
			fmt.Printf("Oda %s baslatildi\n", r.ID)
			r.BroadcastRoomState()

		//oyuncu ayrıldığında
		case client := <-r.Unregister:
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				close(client.send)
				//motor tarafında oyuncu sil
				r.Engine.RemovePlayer(client.Nickname)
				fmt.Printf("Oda %s:Oyuncu ayrıldı\n", r.ID)
				if r.Owner == client.Nickname {
					next := r.Engine.GetPlayerNames()
					if len(next) > 0 {
						r.Owner = next[0]
					} else {
						r.Owner = ""
					}
				}
				r.evaluateGameOver()
				r.BroadcastPlayerList()
				r.BroadcastRoomState()
				if len(r.Clients) == 0 {
					r.closeRoom()
					return
				}
			}
		//bir mesaj geldiğinde
		case message := <-r.Broadcast:
			//Gelen mesajları işle
			r.handleIncomingMessage(message)
		//oyun döngüsü (her TIK ta çalışır)
		case <-ticker.C:
			if r.GameStarted && !r.GameOver {
				//motoru bir adım ilerlet
				r.Engine.Update()
				r.evaluateGameOver()
			}
			//Yeni dünya durumunu herkese gönder
			r.broadcastWorldState()
			if len(r.Clients) == 0 {
				r.closeRoom()
				return
			}

		}
	}
}

func (r *Room) closeRoom() {
	if r.Closed == nil {
		return
	}
	select {
	case r.Closed <- r.ID:
	default:
	}
}

// gelen mesajları Engine e yönlendiren yardımcı fonksiyon
func (r *Room) handleIncomingMessage(rawMessage []byte) {
	var msg GameMessage
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		return
	}

	if msg.Type == "MOVE" {
		if !r.GameStarted || r.GameOver {
			return
		}
		r.Engine.SetPlayerInput(msg.Nickname, PlayerInput{Up: msg.IsMoving})
	} else if msg.Type == "INPUT" {
		if !r.GameStarted || r.GameOver {
			return
		}
		r.Engine.SetPlayerInput(msg.Nickname, PlayerInput{
			Up:    msg.Up,
			Down:  msg.Down,
			Left:  msg.Left,
			Right: msg.Right,
		})
		if msg.Fire {
			r.Engine.TryFire(msg.Nickname)
		}
	} else if msg.Type == "START_GAME" {
		r.tryStartGame(msg.Nickname)
	} else if msg.Type == "FIRE" {
		if !r.GameStarted || r.GameOver {
			return
		}

		r.Engine.TryFire(msg.Nickname)
	}
}

func (r *Room) tryStartGame(requestedBy string) {
	if requestedBy != r.Owner {
		return
	}
	if r.GameStarted && !r.GameOver {
		return
	}
	if r.Engine.PlayerCount() < 2 {
		return
	}

	r.Engine.ResetRound()
	r.GameStarted = true
	r.GameOver = false
	r.Winner = ""

	r.broadcastSimple(map[string]interface{}{
		"type": "GAME_STARTED",
	})
	r.BroadcastRoomState()
}

func (r *Room) evaluateGameOver() {
	if !r.GameStarted || r.GameOver {
		return
	}
	if r.Engine.PlayerCount() <= 1 {
		return
	}

	alive := r.Engine.AlivePlayerNames()
	if len(alive) > 1 {
		return
	}

	r.GameStarted = false
	r.GameOver = true
	if len(alive) == 1 {
		r.Winner = alive[0]
	} else {
		r.Winner = ""
	}

	r.broadcastSimple(map[string]interface{}{
		"type":   "GAME_OVER",
		"winner": r.Winner,
	})
	r.BroadcastRoomState()
}

func (r *Room) broadcastSimple(payload map[string]interface{}) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for client := range r.Clients {
		select {
		case client.send <- raw:
		default:
		}
	}
}

func (r *Room) broadcastWorldState() {
	//engineden güncel verileri al
	state := map[string]interface{}{
		"type":       "GAME_STATE",
		"players":    r.Engine.Players,
		"bullets":    r.Engine.Bullets,
		"started":    r.GameStarted,
		"gameOver":   r.GameOver,
		"winner":     r.Winner,
		"owner":      r.Owner,
		"maxPlayers": MaxPlayersPerRoom,
	}
	raw, _ := json.Marshal(state)

	for client := range r.Clients {
		select {
		case client.send <- raw:
		default:
			//kanal doluysa oyuncuyu temizle
			delete(r.Clients, client)

		}
	}
}

func (r *Room) BroadcastPlayerList() {
	names := r.Engine.GetPlayerNames()
	msg := map[string]interface{}{
		"type":    "PLAYER_LIST",
		"players": names,
	}
	raw, _ := json.Marshal(msg)
	for client := range r.Clients {
		client.send <- raw
	}
}

func (r *Room) BroadcastRoomState() {
	names := r.Engine.GetPlayerNames()
	msg := map[string]interface{}{
		"type":       "ROOM_STATE",
		"roomId":     r.ID,
		"owner":      r.Owner,
		"players":    names,
		"started":    r.GameStarted,
		"gameOver":   r.GameOver,
		"winner":     r.Winner,
		"maxPlayers": MaxPlayersPerRoom,
	}
	raw, _ := json.Marshal(msg)
	for client := range r.Clients {
		client.send <- raw
	}
}
