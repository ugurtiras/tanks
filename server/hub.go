package main

import (
	"crypto/rand"
	"fmt"
	"strings"
)

const roomIDLength = 7

type JoinRoomRequest struct {
	Client   *Client
	Nickname string
	Action   string
	RoomID   string
	Response chan JoinRoomResult
}

type JoinRoomResult struct {
	Room   *Room
	RoomID string
	Err    error
}

type Hub struct {
	Rooms      map[string]*Room //aktif tüm odaların listesi
	JoinRoom   chan JoinRoomRequest
	Unregister chan *Client //sunucudan kopuş
	RoomClosed chan string
}

func newHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		JoinRoom:   make(chan JoinRoomRequest),
		Unregister: make(chan *Client),
		RoomClosed: make(chan string, 8),
	}
}

func (h *Hub) run() {
	for {
		select {
		case req := <-h.JoinRoom:
			h.handleJoinRoomRequest(req)
		case client := <-h.Unregister:
			if client.room != nil {
				client.room.Unregister <- client
				client.room = nil
			}
		case roomID := <-h.RoomClosed:
			delete(h.Rooms, roomID)

		}
	}

}

func (h *Hub) handleJoinRoomRequest(req JoinRoomRequest) {
	switch req.Action {
	case "create":
		roomID := h.generateUniqueRoomID()
		room := NewRoom(roomID, h.RoomClosed)
		room.Owner = req.Nickname
		h.Rooms[roomID] = room
		go room.Run()

		room.Register <- req.Client
		req.Client.room = room
		req.Response <- JoinRoomResult{Room: room, RoomID: roomID}
	case "join":
		roomID := strings.ToUpper(strings.TrimSpace(req.RoomID))
		room := h.Rooms[roomID]
		if room == nil {
			req.Response <- JoinRoomResult{Err: fmt.Errorf("oda bulunamadi")}
			return
		}

		room.Register <- req.Client
		req.Client.room = room
		req.Response <- JoinRoomResult{Room: room, RoomID: roomID}
	default:
		req.Response <- JoinRoomResult{Err: fmt.Errorf("gecersiz oda islemi")}
	}
}

func (h *Hub) generateUniqueRoomID() string {
	for {
		candidate := randomRoomID(roomIDLength)
		if _, exists := h.Rooms[candidate]; !exists {
			return candidate
		}
	}
}

func randomRoomID(length int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buffer := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "AAAAAAA"
	}

	for i := 0; i < length; i++ {
		buffer[i] = alphabet[int(randomBytes[i])%len(alphabet)]
	}

	return string(buffer)
}
