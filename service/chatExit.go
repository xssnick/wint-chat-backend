package service

import (
	"log"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type ExitChat struct {
	*Service
}

func (s *ExitChat) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)

	err := s.Matchmaker.ExitMatch(uid)
	if err != nil {
		log.Println("exit chat:", err)
		return simply.ErrInternal("server error")
	}

	return simply.Empty()
}
