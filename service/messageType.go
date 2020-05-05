package service

import (
	"github.com/valyala/fasthttp"
	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type MessageType struct {
	*Service
}

type MessageTypeStop struct {
	*Service
}

type messageTypeRequest struct {
	IsGame bool `json:"is_game"`
	On uint64 `json:"on"`
}

func (l *MessageType) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.Json(messageTypeRequest{}, req.Body())
}

func (l *MessageTypeStop) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.Json(messageTypeRequest{}, req.Body())
}

func (l *MessageType) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(messageTypeRequest)

	uid := flow.Context().Value("user_id").(uint64)

	if !req.IsGame {
		req.On = chatHash(req.On,uid)
	}

	l.Typer.SetType(req.On, uid)

	return simply.Empty()
}

func (l *MessageTypeStop) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(messageTypeRequest)

	uid := flow.Context().Value("user_id").(uint64)

	if !req.IsGame {
		req.On = chatHash(req.On,uid)
	}

	l.Typer.DelType(req.On, uid)

	return simply.Empty()
}
