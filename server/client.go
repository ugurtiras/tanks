package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

//Goroutine yapısı

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	room     *Room
	Nickname string
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type GameMessage struct {
	Type     string  `json:"type"`
	Nickname string  `json:"nickname"`
	Action   string  `json:"action"`
	RoomID   string  `json:"roomId"`
	IsMoving bool    `json:"isMoving"` // client'dan gelen hareket durumu
	Angle    float64 `json:"angle"`    // Tankın veya merminin açısı
	Up       bool    `json:"up"`
	Down     bool    `json:"down"`
	Left     bool    `json:"left"`
	Right    bool    `json:"right"`
	Fire     bool    `json:"fire"`
	Message  string  `json:"message"`
}

var nicknameRegex = regexp.MustCompile(`^[A-Za-z0-9_]{3,12}$`)
var roomIDRegex = regexp.MustCompile(`^[A-Z]{7}$`)

func (c *Client) sendJSON(payload map[string]interface{}) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	c.send <- raw
}

func (c *Client) sendAuthFail(message string) {
	c.sendJSON(map[string]interface{}{
		"type":    "AUTH_FAIL",
		"message": message,
	})
}

func normalizeNickname(raw string) string {
	return strings.TrimSpace(raw)
}

func normalizeRoomID(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func (c *Client) handleLogin(incomingMsg GameMessage) {
	if c.Nickname != "" {
		c.sendAuthFail("Zaten giris yaptin.")
		return
	}

	nickname := normalizeNickname(incomingMsg.Nickname)
	if !nicknameRegex.MatchString(nickname) {
		c.sendAuthFail("Nick 3-12 karakter olmali (harf/rakam/_).")
		return
	}

	action := strings.ToLower(strings.TrimSpace(incomingMsg.Action))
	roomID := normalizeRoomID(incomingMsg.RoomID)

	joinReq := JoinRoomRequest{
		Client:   c,
		Nickname: nickname,
		Action:   action,
		RoomID:   roomID,
		Response: make(chan JoinRoomResult, 1),
	}

	if action == "join" && !roomIDRegex.MatchString(roomID) {
		c.sendAuthFail("Oda kodu 7 haneli buyuk harf olmali.")
		return
	}

	if action != "join" && action != "create" {
		c.sendAuthFail("Gecersiz istek. Odaya katil veya oda olustur sec.")
		return
	}

	c.hub.JoinRoom <- joinReq
	joinRes := <-joinReq.Response
	if joinRes.Err != nil {
		c.sendAuthFail(joinRes.Err.Error())
		return
	}

	c.room = joinRes.Room
	c.Nickname = nickname

	if err := c.room.Engine.AddPlayer(c.Nickname); err != nil {
		if err == ErrRoomFull {
			c.sendAuthFail("Oda dolu. Maksimum 4 oyuncu.")
		} else {
			c.sendAuthFail("Bu nick bu odada zaten kullanimda.")
		}
		c.hub.Unregister <- c
		c.Nickname = ""
		return
	}

	fmt.Printf("Oyuncu LOGIN oldu: %s (Oda: %s)\n", c.Nickname, joinRes.RoomID)
	c.room.BroadcastPlayerList()
	c.room.BroadcastRoomState()
	c.sendJSON(map[string]interface{}{
		"type":     "AUTH_SUCCESS",
		"nickname": c.Nickname,
		"roomId":   joinRes.RoomID,
		"isOwner":  c.room.Owner == c.Nickname,
	})
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister <- c //unregister yaparak hafıza temizliği yapıyoruz
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)              //büyük paketleri alma
	c.conn.SetReadDeadline(time.Now().Add(pongWait)) //zaman aşımı ayarı
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait)) //oyuncu cevap verirse süreyi uzat (60 saniye)
		return nil
	})
	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var incomingMsg GameMessage
		if err := json.Unmarshal(rawMessage, &incomingMsg); err != nil {
			continue
		}

		switch incomingMsg.Type {
		case "LOGIN":
			c.handleLogin(incomingMsg)

		case "START_GAME":
			if c.room != nil {
				incomingMsg.Nickname = c.Nickname
				normalized, err := json.Marshal(incomingMsg)
				if err != nil {
					continue
				}
				c.room.Broadcast <- normalized
			}

		default:
			if c.room != nil {
				incomingMsg.Nickname = c.Nickname
				incomingMsg.RoomID = ""
				incomingMsg.Action = ""
				normalized, err := json.Marshal(incomingMsg)
				if err != nil {
					continue
				}
				c.room.Broadcast <- normalized
			}
		}
	}

}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod) //düzenli aralıklarla ping atan saat
	defer func() {

		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send: //hubdan bu oyuncuya mesaj gelirse onu websocket kanalına fırlat
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok { //kanal kapandıysa bağlantıyı kapat
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return

			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C: //ticker zamanı geldiyse oyuncuya ping at
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return

			}
		}
	}
}
