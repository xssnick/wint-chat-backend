package service

import (
	"log"

	"github.com/valyala/fasthttp"
	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"

	"github.com/xssnick/wint/pkg/repo"
)

type StateChat struct {
	*Service
}

func (s *StateChat) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	type stateChatResponse struct {
		ID         uint64 `json:"id"`
		Players    int    `json:"players"`
		InProgress bool   `json:"in_progress"`
		InSearch   bool   `json:"in_search"`
	}

	uid := flow.Context().Value("user_id").(uint64)

	state, err := s.Matchmaker.StateMatch(uid, true)
	if err != nil {
		if err == repo.ErrNotFound {
			return stateChatResponse{
				InSearch:   false,
				InProgress: false,
			}
		}
		log.Println("state chat:", err)
		return simply.ErrInternal("server error")
	}

	log.Println(state)

	return stateChatResponse{
		ID:         state.ID,
		InSearch:   state.StartedAt == 0,
		Players:    len(state.Players),
		InProgress: state.StartedAt > 0 && state.FinishedAt == 0,
	}
}

func (s *StateChat) OnHttpResponse(flow goeasy.Flow, result interface{}, resp *fasthttp.Response) {
	log.Println(result)
	s.Service.OnHttpResponse(flow, result, resp)
}
