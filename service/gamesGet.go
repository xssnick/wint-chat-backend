package service

import (
	"encoding/binary"
	"encoding/hex"
	"log"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type GamesGet struct {
	*Service
}

func (l *GamesGet) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)

	games, err := l.Repo.GetGames(flow.Context(), uid)
	if err != nil {
		log.Println("chats get err:", uid, err)
		return simply.ErrInternal("internal error")
	}

	type player struct {
		ID uint64 `json:"id"`
		IsWinner bool `json:"is_winner"`
		IsOwner bool `json:"is_owner"`
		Avatar string `json:"avatar"`
		Name string `json:"name"`
	}

	type game struct {
		ID uint64 `json:"id"`
		Players []player `json:"players"`
		Time int64 `json:"time"`
	}

	res := make([]game, 0, len(games))
	for _, g := range games {
		gm := game{
			ID:      g.ID,
			Players: make([]player, 0, len(g.Players)),
			Time:    g.FinishedAt.Unix(),
		}

		for _, p := range g.Players {
			bts := make([]byte, 8)
			binary.LittleEndian.PutUint64(bts, p.ID)

			enc, err := l.Cryptor.Encrypt(bts)
			if err != nil {
				log.Println("profile id enc error:", err)
				return simply.ErrInternal("internal error")
			}

			gm.Players = append(gm.Players, player{
				ID:     p.ID,
				Name: p.Name,
				Avatar: "/images/" + hex.EncodeToString(enc) + "/0.jpg",
				IsWinner: p.IsWinner,
				IsOwner: p.IsOwner,
			})
		}

		res = append(res, gm)
	}

	return res
}
