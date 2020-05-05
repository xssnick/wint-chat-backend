package service

import (
	"log"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type CreateChat struct {
	*Service
}

func (s *CreateChat) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)

	user, err := s.Repo.GetUserProfile(flow.Context(), uid)
	if err != nil {
		log.Println("create_chat GetUserProfile:", err)
		return simply.ErrInternal("server error")
	}

	err = s.Matchmaker.CreateMatch(uid, user.Mode)
	if err != nil {
		log.Println("create chat:", err)
		return simply.ErrInternal("server error")
	}

	return simply.Empty()
}
