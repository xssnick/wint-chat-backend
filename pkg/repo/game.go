package repo

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrNotEnoughPlayers = errors.New("not enough players")
var ErrAlreadyInSearch = errors.New("search is already in progress")

type Player struct {
	ID       uint64
	Name     string
	IsWinner bool
	IsOwner bool
}

type Game struct {
	ID         uint64
	FinishedAt time.Time
	Players    []Player
}

type MatchState struct {
	ID         uint64
	Owner      uint64
	Players    []uint64
	StartedAt  int64
	FinishedAt int64
}

type Message struct {
	ID        uint64
	Name      string
	Owner     uint64
	Text      string
	CreatedAt int64
}

type Like struct {
	MessageID uint64
	UserID    uint64
}

type Chat struct {
	ID uint64
	IsSame bool
	MessageOwner uint64
	Message string
	Unread int
	MessageTime time.Time
}