package service

import (
	"log"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type FindChat struct {
	*Service
}

func (s *FindChat) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)

	user, err := s.Repo.GetUserProfile(flow.Context(), uid)
	if err != nil {
		log.Println("create_chat GetUserProfile:", err)
		return simply.ErrInternal("server error")
	}

	mode := user.Mode
	if mode != 0 {
		mode = user.Sex
	}

	err = s.Matchmaker.FindMatch(uid, mode)
	if err != nil {
		log.Println("find chat:", err)
		return simply.ErrInternal("server error")
	}

	return simply.Empty()
}
