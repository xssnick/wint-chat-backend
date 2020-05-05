package service

import (
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"strconv"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"

	"github.com/xssnick/wint/pkg/repo"
)

type Login struct {
	*Service
}

type loginRequest struct {
	Phone uint64 `json:"phone,string"`
}

func (l *Login) MiddlewareChain() []goeasy.HttpMiddleware {
	return []goeasy.HttpMiddleware{}
}

func (l *Login) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.Json(loginRequest{}, req.Body())
}

func (l *Login) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(loginRequest)

	log.Println(req.Phone)
	if req.Phone < uint64(math.Pow10(9)) {
		return simply.ErrBadRequest("len not 10")
	}

	long, short := l.OTP.Generate(strconv.FormatUint(req.Phone, 16), 6)

	err := l.Notifier.Notify(req.Phone, short)
	if err != nil {
		log.Println("login sms send failed", err)
		return simply.ErrInternal("sms send failed")
	}

	return simply.Dynamic{
		"auth_code": long,
	}
}

type LoginEnd struct {
	*Service
}

type loginEndRequest struct {
	Phone      uint64 `json:"phone,string"`
	NotifyCode string `json:"notify_code"`
	AuthCode   []byte `json:"auth_code"`
}

func (l *LoginEnd) MiddlewareChain() []goeasy.HttpMiddleware {
	return []goeasy.HttpMiddleware{}
}

func (l *LoginEnd) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.Json(loginEndRequest{}, req.Body())
}

func (l *LoginEnd) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(loginEndRequest)

	if req.Phone < uint64(math.Pow10(9)) {
		return simply.ErrBadRequest("len not 10")
	}

	if !l.OTP.Validate(strconv.FormatUint(req.Phone, 16), 6, req.AuthCode, req.NotifyCode) {
		return simply.ErrAccessDenied("incorrect code")
	}

	// TODO: attempts

	isnew := false
	id, err := l.Repo.GetUserIDByPhone(flow.Context(), req.Phone)
	if err != nil {
		if err == repo.ErrNotFound {
			id, err = l.Repo.CreateUser(flow.Context(), req.Phone)
			isnew = true
		}

		if err != nil {
			log.Println("login error on get/create user", err)
			return simply.ErrInternal("internal login error")
		}
	}

	if !isnew {
		profile, err := l.Repo.GetUserProfile(flow.Context(), id)
		if err != nil {
			log.Println("login error on get user profile", err)
			return simply.ErrInternal("internal login error")
		}

		if !profile.RegFinished {
			isnew = true
		}
	}

	// TODO: session id

	token := make([]byte, 16)
	binary.LittleEndian.PutUint64(token, id)
	binary.LittleEndian.PutUint64(token[8:], rand.Uint64())

	enc, err := l.Cryptor.Encrypt(token)
	if err != nil {
		log.Println("login error on get/create user", err)
		return simply.ErrInternal("token creation error")
	}

	return simply.Dynamic{
		"id":    id,
		"token": enc,
		"isnew": isnew,
	}
}
