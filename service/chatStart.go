package service

import (
	"log"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type StartChat struct {
	*Service
}

func (s *StartChat) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)

	err := s.Matchmaker.StartMatch(uid)
	if err != nil {
		log.Println("start chat:", err)
		return simply.ErrInternal("server error")
	}

	return simply.Empty()
}
