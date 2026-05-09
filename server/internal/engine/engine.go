package engine

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

var ErrRoomFull = errors.New("oda dolu")
var ErrNicknameTaken = errors.New("bu nick bu odada zaten kullanimda")

const MaxPlayersPerRoom = 4

const (
	TileSize   = 40.0
	Rows       = 20
	Cols       = 20
	mapWidth   = float64(Cols * TileSize) //800
	mapHeight  = float64(Cols * TileSize) //800
	TankRadius = 12.0
)

var MazeData = [Rows][Cols]int{
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1},
	{1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 1},
	{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

type GameEngine struct {
	mu      sync.RWMutex
	Players map[string]*PlayerState
	Bullets []*BulletState
}

type PlayerState struct {
	X          float64     `json:"x"`
	Y          float64     `json:"y"`
	Angle      float64     `json:"angle"`
	Input      PlayerInput `json:"-"`
	LastShotAt time.Time   `json:"-"`
	Health     int         `json:"health"`
}

type PlayerInput struct {
	Up    bool
	Down  bool
	Left  bool
	Right bool
}

type BulletState struct {
	ID        string    `json:"id"`
	Owner     string    `json:"owner"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	Angle     float64   `json:"angle"`
	Alive     bool      `json:"-"`
	CreatedAt time.Time `json:"-"`
}

type PlayerObservation struct {
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Angle  float64 `json:"angle"`
	Health int     `json:"health"`
	Up     bool    `json:"up"`
	Down   bool    `json:"down"`
	Left   bool    `json:"left"`
	Right  bool    `json:"right"`
}

type BulletObservation struct {
	Owner string  `json:"owner"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Angle float64 `json:"angle"`
	Alive bool    `json:"alive"`
}

type Observation struct {
	PlayerCount int                 `json:"playerCount"`
	BulletCount int                 `json:"bulletCount"`
	Players     []PlayerObservation `json:"players"`
	Bullets     []BulletObservation `json:"bullets"`
}

func isWall(x, y float64) bool {
	if x < 0 || y < 0 {
		return true
	}

	gridX := int(x / TileSize)
	gridY := int(y / TileSize)
	if gridY < 0 || gridY >= Rows || gridX < 0 || gridX >= Cols {
		return true
	}

	return MazeData[gridY][gridX] == 1
}

func collidesWallWithRadius(x, y, radius float64) bool {

	return isWall(x-radius, y-radius) ||
		isWall(x+radius, y-radius) ||
		isWall(x-radius, y+radius) ||
		isWall(x+radius, y+radius)
}

func NewGameEngine() *GameEngine {
	return &GameEngine{
		Players: make(map[string]*PlayerState),
		Bullets: make([]*BulletState, 0),
	}
}

func (e *GameEngine) AddPlayer(nickname string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.Players) >= MaxPlayersPerRoom {
		return ErrRoomFull
	}
	if _, exists := e.Players[nickname]; exists {
		return ErrNicknameTaken
	}
	e.Players[nickname] = &PlayerState{
		X:          TileSize * 1.5,
		Y:          TileSize * 1.5,
		Angle:      0,
		LastShotAt: time.Now().Add(-time.Second),
		Health:     100,
	}
	return nil
}

func (e *GameEngine) RemovePlayer(nickname string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.Players, nickname)
}

func (e *GameEngine) PlayerCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.Players)
}

func (e *GameEngine) GetPlayerNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.Players))
	for name := range e.Players {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (e *GameEngine) AlivePlayerNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.Players))
	for name, player := range e.Players {
		if player.Health > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (e *GameEngine) ResetRound() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.Bullets = make([]*BulletState, 0)
	names := make([]string, 0, len(e.Players))
	for name := range e.Players {
		names = append(names, name)
	}
	sort.Strings(names)

	spawnPositions := []struct {
		x float64
		y float64
	}{
		{TileSize * 1.5, TileSize * 1.5},
		{TileSize * 18.5, TileSize * 1.5},
		{TileSize * 1.5, TileSize * 18.5},
		{TileSize * 18.5, TileSize * 18.5},
	}

	for i, name := range names {
		p := e.Players[name]
		spawn := spawnPositions[i%len(spawnPositions)]
		p.X = spawn.x
		p.Y = spawn.y
		p.Angle = 0
		p.Input = PlayerInput{}
		p.Health = 100
		p.LastShotAt = time.Now().Add(-time.Second)
	}
}

// her Tick'te çalışacak ana motor fonksiyonu. dt saniye cinsinden zaman farkı
func (e *GameEngine) Update(dt float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	//Oyuncuları hareket ettir
	moveSpeed := 200.0 // units per second
	turnSpeed := 8.0   // radians per second
	for _, p := range e.Players {
		if p.Health <= 0 {
			continue
		}

		if p.Input.Left {
			p.Angle -= turnSpeed * dt
		}
		if p.Input.Right {
			p.Angle += turnSpeed * dt
		}

		forward := 0.0
		if p.Input.Up {
			forward += 1.0
		}
		if p.Input.Down {
			forward -= 1.0
		}

		if forward != 0 {
			nextX := p.X + math.Cos(p.Angle)*moveSpeed*forward*dt
			nextY := p.Y + math.Sin(p.Angle)*moveSpeed*forward*dt

			if !collidesWallWithRadius(nextX, p.Y, TankRadius) {
				p.X = nextX
			}
			if !collidesWallWithRadius(p.X, nextY, TankRadius) {
				p.Y = nextY
			}

			if p.X < TankRadius {
				p.X = TankRadius
			} else if p.X > mapWidth-TankRadius {
				p.X = mapWidth - TankRadius
			}
			if p.Y < TankRadius {
				p.Y = TankRadius
			} else if p.Y > mapHeight-TankRadius {
				p.Y = mapHeight - TankRadius
			}
		}
	}

	e.updateBullets(dt)
}

func (e *GameEngine) updateBullets(dt float64) {
	bulletSpeed := 600.0 // units per second
	hitDistance := 20.0

	activeBullets := make([]*BulletState, 0)

	for _, b := range e.Bullets {
		// 1. ZAMAN KONTROLÜ (2 Saniye Kuralı)
		if time.Since(b.CreatedAt) > 2*time.Second {
			b.Alive = false
			continue
		}

		steps := 3.0
		stepSpeed := (bulletSpeed * dt) / steps
		for i := 0; i < int(steps); i++ {
			dx := math.Cos(b.Angle) * stepSpeed
			dy := math.Sin(b.Angle) * stepSpeed

			nextX := b.X + dx
			nextY := b.Y + dy

			hitXWall := isWall(nextX, b.Y)
			hitYWall := isWall(b.X, nextY)

			if hitXWall && hitYWall {
				b.Angle += math.Pi
				continue
			}

			if hitXWall {
				b.Angle = math.Pi - b.Angle
			} else {
				b.X = nextX
			}

			if hitYWall {
				b.Angle = -b.Angle
			} else {
				b.Y = nextY
			}

			if b.X <= 1 {
				b.X = 1
				b.Angle = math.Pi - b.Angle
			} else if b.X >= mapWidth-1 {
				b.X = mapWidth - 1
				b.Angle = math.Pi - b.Angle
			}
			if b.Y <= 1 {
				b.Y = 1
				b.Angle = -b.Angle
			} else if b.Y >= mapHeight-1 {
				b.Y = mapHeight - 1
				b.Angle = -b.Angle
			}
		}

		hit := false
		for nick, p := range e.Players {
			if nick == b.Owner && time.Since(b.CreatedAt) < 200*time.Millisecond {
				continue // Mermi namludan yeni çıktıysa sahibini vurmasın
			}

			dist := math.Sqrt(math.Pow(p.X-b.X, 2) + math.Pow(p.Y-b.Y, 2))
			if dist < hitDistance {
				p.Health = 0 // TEK ATIŞTA ÖLÜM!(sonrasında güncelleyebilmek için)
				b.Alive = false
				hit = true
				fmt.Printf(" %s, %s tarafından YOK EDİLDİ!\n", nick, b.Owner)
				break
			}
		}

		if !hit && b.Alive {
			activeBullets = append(activeBullets, b)
		}
	}
	e.Bullets = activeBullets
}

func (e *GameEngine) SetPlayerInput(nickname string, input PlayerInput) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p, ok := e.Players[nickname]; ok {
		p.Input = input
	}
}

func (e *GameEngine) TryFire(nickname string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.Players[nickname]
	if !ok || p.Health <= 0 {
		return
	}

	if time.Since(p.LastShotAt) < 250*time.Millisecond {
		return
	}

	p.LastShotAt = time.Now()
	e.addBulletLocked(nickname, p.X, p.Y, p.Angle)
}

// Yeni bir mermi ateşlendiğinde listeye ekler
func (e *GameEngine) AddBullet(nickname string, x, y, angle float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.addBulletLocked(nickname, x, y, angle)
}

func (e *GameEngine) addBulletLocked(nickname string, x, y, angle float64) {

	newBullet := &BulletState{
		ID:        fmt.Sprintf("%s-%d", nickname, time.Now().UnixNano()),
		Owner:     nickname,
		X:         x,
		Y:         y,
		Angle:     angle,
		Alive:     true,
		CreatedAt: time.Now(),
	}
	e.Bullets = append(e.Bullets, newBullet)
}

func (g *GameEngine) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Bullets = make([]*BulletState, 0)

	names := make([]string, 0, len(g.Players))
	for name := range g.Players {
		names = append(names, name)
	}
	sort.Strings(names)

	spawnPositions := []struct {
		x float64
		y float64
	}{
		{TileSize * 1.5, TileSize * 1.5},
		{TileSize * 18.5, TileSize * 1.5},
		{TileSize * 1.5, TileSize * 18.5},
		{TileSize * 18.5, TileSize * 18.5},
	}

	for i, name := range names {
		player := g.Players[name]
		spawn := spawnPositions[i%len(spawnPositions)]
		player.X = spawn.x
		player.Y = spawn.y
		player.Angle = 0
		player.Input = PlayerInput{}
		player.LastShotAt = time.Now().Add(-time.Second)
		player.Health = 100
	}
}

func (g *GameEngine) GameState() Observation {
	g.mu.RLock()
	defer g.mu.RUnlock()

	obs := Observation{
		PlayerCount: len(g.Players),
		BulletCount: len(g.Bullets),
		Players:     make([]PlayerObservation, 0, len(g.Players)),
		Bullets:     make([]BulletObservation, 0, len(g.Bullets)),
	}

	names := make([]string, 0, len(g.Players))
	for name := range g.Players {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		player := g.Players[name]

		obs.Players = append(obs.Players, PlayerObservation{
			Name:   name,
			X:      player.X,
			Y:      player.Y,
			Angle:  player.Angle,
			Health: player.Health,
			Up:     player.Input.Up,
			Down:   player.Input.Down,
			Left:   player.Input.Left,
			Right:  player.Input.Right,
		})
	}

	for _, bullet := range g.Bullets {
		obs.Bullets = append(obs.Bullets, BulletObservation{
			Owner: bullet.Owner,
			X:     bullet.X,
			Y:     bullet.Y,
			Angle: bullet.Angle,
			Alive: bullet.Alive,
		})
	}

	return obs
}
