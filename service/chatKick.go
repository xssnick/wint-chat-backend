package service

import (
	"log"

	"github.com/valyala/fasthttp"
	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type KickChat struct {
	*Service
}

type kickChatRequest struct {
	Player uint64 `json:"player"`
	Game   uint64 `json:"game"`
}

func (s *KickChat) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.JsonPtr(new(kickChatRequest), req.Body())
}

func (s *KickChat) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)
	req := p.(*kickChatRequest)

	mstate, err := s.Repo.GetGameStateByID(flow.Context(), req.Game, uid, false)
	if err != nil {
		log.Println("kick chat get game:", err)
		return simply.ErrInternal("server error")
	}

	if mstate.FinishedAt > 0 {
		return simply.ErrAccessDenied("already finished")
	}

	if uid == mstate.Owner && req.Player == uid {
		err = s.Repo.FinishGame(flow.Context(), uid, req.Game, 0)
		if err != nil {
			log.Println("kick chat finish game:", err)
			return simply.ErrInternal("server error")
		}

		s.Poller.PushEvent(mstate.ID)
		return simply.Empty()
	}

	if uid != mstate.Owner && req.Player == uid {
		err = s.Repo.KickFromGame(flow.Context(), mstate.Owner, req.Game, uid)
		if err != nil {
			log.Println("kick chat as owner:", err)
			return simply.ErrInternal("server error")
		}

		s.Poller.PushEvent(mstate.ID)
		return simply.Empty()
	}

	err = s.Repo.KickFromGame(flow.Context(), uid, req.Game, req.Player)
	if err != nil {
		log.Println("kick chat:", err)
		return simply.ErrInternal("server error")
	}

	s.Poller.PushEvent(mstate.ID)

	return simply.Empty()
}
