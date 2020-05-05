package service

import (
	"log"

	"github.com/valyala/fasthttp"
	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"

	"github.com/xssnick/wint/pkg/repo"
)

type MessageLike struct {
	*Service
}

type messageLikeRequest struct {
	Delete bool   `json:"delete"`
	ID     uint64 `json:"id"`
}

func (l *MessageLike) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.Json(messageLikeRequest{}, req.Body())
}

func (l *MessageLike) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(messageLikeRequest)

	log.Println("WANTLIKE")

	uid := flow.Context().Value("user_id").(uint64)

	mstate, err := l.Matchmaker.StateMatch(uid, true)
	if err != nil {
		if err == repo.ErrNotFound {
			return simply.ErrNotFound("no active game")
		}

		log.Println("get state for msg like to game err,:", uid, err)
		return simply.ErrInternal("internal error")
	}

	if mstate.FinishedAt > 0 {
		return simply.ErrNotFound("no active game, already finished")
	}

	if mstate.StartedAt == 0 {
		return simply.ErrNotFound("no active game, not started")
	}

	msgs, err := l.Repo.GetMessagesGame(flow.Context(), mstate.ID)
	if err != nil {
		log.Println("msg get of game err:", mstate.ID, uid, err)
		return simply.ErrInternal("internal error")
	}

	var found bool
	for _, m := range msgs {
		if m.ID == req.ID && m.Owner != uid {
			found = true
			break
		}
	}

	if !found {
		return simply.ErrNotFound("message not found")
	}

	likes, err := l.Repo.GetLikesGame(flow.Context(), mstate.ID)
	if err != nil {
		log.Println("likes get of game err:", mstate.ID, uid, err)
		return simply.ErrInternal("internal error")
	}

	/*var last uint64
	if len(msgs) > 0 {
		last = msgs[len(msgs)-1].ID
	}*/

	log.Println("DOLIKE")

	for _, lk := range likes {
		if lk.UserID == uid && lk.MessageID == req.ID {
			if !req.Delete {
				return simply.ErrAccessDenied("already liked")
			}

			err := l.Repo.DeleteLike(flow.Context(), uid, req.ID)
			if err != nil {
				log.Println("msg like del of game err:", mstate.ID, uid, req.ID, err)
				return simply.ErrInternal("internal error")
			}

			l.Poller.PushEvent(mstate.ID)

			return simply.Empty()
		}
	}

	if req.Delete {
		return simply.ErrNotFound("message not found to delete")
	}

	err = l.Repo.SetLike(flow.Context(), uid, req.ID)
	if err != nil {
		log.Println("msg like of game err:", mstate.ID, uid, req.ID, err)
		return simply.ErrInternal("internal error")
	}

	l.Poller.PushEvent(mstate.ID)

	return simply.Empty()
}
