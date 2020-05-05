package service

import (
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"

	"github.com/xssnick/wint/pkg/repo"
)

type MessagesGet struct {
	*Service
}

type messagesGetRequest struct {
	IsGame bool
	Force bool
	ID     uint64
	Hash   uint64
}

func (l *MessagesGet) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	id, err := req.URI().QueryArgs().GetUint("id")
	if err != nil {
		return simply.ErrBadRequest("id not  valid")
	}

	hash, err := req.URI().QueryArgs().GetUint("hash")
	return messagesGetRequest{
		Hash:   uint64(hash),
		Force: err != nil,
		ID:     uint64(id),
		IsGame: req.URI().QueryArgs().GetBool("isgame"),
	}
}

func (l *MessagesGet) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(messagesGetRequest)

	uid := flow.Context().Value("user_id").(uint64)

	log.Println("START",uid, req.Hash,req.Force)
	defer log.Println("STOP",uid,req.Hash,req.Force)

	if req.IsGame {
		member, err := l.Repo.IsGameMember(flow.Context(), req.ID, uid)
		if err != nil {
			log.Println("get member for msg send to game err:", uid, err)
			return simply.ErrInternal("internal error")
		}

		if !member {
			return simply.ErrAccessDenied("not your game")
		}

		var found bool
		if !req.Force {
			var ch chan bool
			ch, found = l.Poller.WaitEvent(req.Hash, req.ID)

			select {
			case <-ch:
			case <-time.After(10 * time.Second):
				l.Poller.ForgotEvent(req.ID, ch)
				return simply.Dynamic{
					"retry": true,
				}
			}
		} else {
			found = l.Poller.GetEvent(req.ID) != ""
		}

		mstate, err := l.Repo.GetGameStateByID(flow.Context(), req.ID, uid, false)
		if err != nil {
			if err == repo.ErrNotFound {
				return simply.ErrNotFound("no active game")
			}

			log.Println("get state for msg get to game err,:", uid, err)
			return simply.ErrInternal("internal error")
		}

		msgs, err := l.Repo.GetMessagesGame(flow.Context(), mstate.ID)
		if err != nil {
			log.Println("msg get of game err:", mstate.ID, uid, err)
			return simply.ErrInternal("internal error")
		}

		likes, err := l.Repo.GetLikesGame(flow.Context(), mstate.ID)
		if err != nil {
			log.Println("likes get of game err:", mstate.ID, uid, err)
			return simply.ErrInternal("internal error")
		}

		var result = make([]struct {
			ID    uint64 `json:"id"`
			Owner uint64 `json:"owner"`
			Likes uint64 `json:"likes"`
			ILike bool   `json:"i_like"`
			Name  string `json:"name"`
			Text  string `json:"text"`
			Time  int64  `json:"time"`
		}, len(msgs))

		canWrite, myTurn, canDelete, roundStartTime := l.Gaming.CanWriteLeft(uid, &mstate, msgs)

		for i, v := range msgs {
			for _, lv := range likes {
				if lv.MessageID == v.ID {
					result[i].Likes++

					if lv.UserID == uid {
						result[i].ILike = true
					}
				}
			}

			result[i].ID = v.ID
			result[i].Owner = v.Owner
			result[i].Text = v.Text
			result[i].Time = v.CreatedAt
			result[i].Name = v.Name
		}

		pls := append(mstate.Players, mstate.Owner)
		for i, p := range pls {
			if p == uid {
				pls[i] = 0
			}
		}

		typing, err := l.Repo.GetUsersNames(flow.Context(), l.Typer.GetTypes(mstate.ID, pls))
		if err != nil {
			log.Println("msg get of game err:", mstate.ID, uid, err)
			return simply.ErrInternal("internal error")
		}

		var typstr string
		for i, v := range typing {
			typstr += v.Name
			if i == len(typing)-2 {
				typstr += " и "
			} else if i != len(typing)-1 {
				typstr += ", "
			}
		}

		orZero := func(i int64) int64 {
			if i < 0 {
				return 0
			}
			return i
		}

		/*var last uint64
		if len(msgs) > 0 {
			last = msgs[len(msgs)-1].ID
		}*/

		// strconv.FormatUint(last, 16) + "." + strconv.Itoa(len(mstate.Players)) + "." + strconv.FormatInt(mstate.FinishedAt, 16)

		hash := l.Poller.GetEvent(req.ID)

		if !found {
			l.Poller.PushEvent(req.ID)
		}

		stime := time.Now().UTC().UnixNano()/int64(time.Millisecond)

		return simply.Dynamic{
			"hash":       hash,
			"canWrite":   canWrite,
			"canDelete":  canDelete,
			"myTurn":     myTurn,
			"serverTime": stime,
			"roundEnd":   orZero((roundStartTime + 120)*1000 - stime),
			"gameOwner":  mstate.Owner,
			"winner":     mstate.Owner,
			"isFinished": mstate.FinishedAt > 0,
			"players":    mstate.Players,
			"typing":     typstr,
			"messages":   result,
		}
	}

	ch, found := l.Poller.WaitEvent(req.Hash, chatHash(uid, req.ID))

	if !req.Force {
		select {
		case <-ch:
		case <-time.After(10 * time.Second):
			l.Poller.ForgotEvent(chatHash(uid, req.ID), ch)
			return simply.Dynamic{
				"retry": true,
			}
		}
	} else {
		found = l.Poller.GetEvent(chatHash(uid, req.ID)) != ""
	}

	msgs, err := l.Repo.GetMessagesChat(flow.Context(), uid, req.ID)
	if err != nil {
		log.Println("msg get of chat err:", req.ID, uid, err)
		return simply.ErrInternal("internal error")
	}

	err = l.Repo.SetRead(flow.Context(), uid, req.ID)
	if err != nil {
		log.Println("set read of chat err:", req.ID, uid, err)
		return simply.ErrInternal("internal error")
	}

	var result = make([]struct {
		ID    uint64 `json:"id"`
		Owner uint64 `json:"owner"`
		Name  string `json:"name"`
		Text  string `json:"text"`
		Time  int64  `json:"time"`
	}, len(msgs))

	for i, v := range msgs {
		result[i].ID = v.ID
		result[i].Owner = v.Owner
		result[i].Text = v.Text
		result[i].Time = v.CreatedAt
		result[i].Name = v.Name
	}

	unames, err := l.Repo.GetUsersNames(flow.Context(), []uint64{req.ID})
	if err != nil {
		log.Println("msg get of chat typ err:", req.ID, uid, err)
		return simply.ErrInternal("internal error")
	}

	var name string
	if len(unames) > 0 {
		name = unames[0].Name
	}

	typing := l.Typer.GetTypes(chatHash(uid, req.ID), []uint64{req.ID})

	var typstr string
	if len(typing) > 0 {
		typstr = name
	}

	/*var last uint64
	if len(msgs) > 0 {
		last = msgs[len(msgs)-1].ID
	}*/

	// hash := strconv.FormatUint(last, 16)

	hash := l.Poller.GetEvent(chatHash(uid, req.ID))

	if !found {
		l.Poller.PushEvent(chatHash(uid, req.ID))
	}

	log.Println(hash,chatHash(uid, req.ID))

	return simply.Dynamic{
		"hash":        hash,
		"typing":      typstr,
		"partnerName": name,
		"messages":    result,
	}
}
