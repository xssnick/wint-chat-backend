package service

import (
	"log"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"

	"github.com/xssnick/wint/pkg/repo"
)

type MessageSend struct {
	*Service
}

type messageSendRequest struct {
	ToUser uint64 `json:"user"`
	Text   string `json:"text"`
}

func (l *MessageSend) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.Json(messageSendRequest{}, req.Body())
}

func (l *MessageSend) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(messageSendRequest)

	uid := flow.Context().Value("user_id").(uint64)

	var mid uint64
	if req.ToUser == 0 {
		mstate, err := l.Matchmaker.StateMatch(uid, true)
		if err != nil {
			if err == repo.ErrNotFound {
				return simply.ErrNotFound("no active game")
			}

			log.Println("get state for msg send to game err,:", uid, err)
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

		canWrite, _, _, _ := l.Gaming.CanWriteLeft(uid, &mstate, msgs)

		if !canWrite {
			return simply.ErrAccessDenied("already answered")
		}

		mid, err = l.Repo.SendMessage(flow.Context(), uid, true, mstate.ID, req.Text)
		if err != nil {
			log.Println("msg send to game err:", uid, err)
			return simply.ErrInternal("internal error")
		}

		l.Typer.DelType(mstate.ID, uid)

		l.Poller.PushEvent(mstate.ID)
	} else {
		var err error

		mid, err = l.Repo.SendMessage(flow.Context(), uid, false, req.ToUser, req.Text)
		if err != nil {
			log.Println("msg send to user err:", uid, err)
			return simply.ErrInternal("internal error")
		}

		l.Typer.DelType(chatHash(uid, req.ToUser), uid)

		l.Poller.PushEvent(chatHash(uid, req.ToUser))
	}

	log.Println("SENT",mid)

	return simply.Dynamic{
		"message_id": mid,
	}
}

func chatHash(f, s uint64) uint64 {
	if f > s {
		return f | (s << 32)
	}
	return s | (f << 32)
}
