package service

import (
	"context"
	"encoding/base64"
	"encoding/binary"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type Auth struct {
	*Service
}

type authRequest struct {
	Token []byte
}

func (s *Auth) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	/*val, err := url.QueryUnescape(string(req.Header.Peek("Auth-Token")))
	if err != nil {
		return simply.ErrUnauthorized("unauthorized")
	}*/

	token, err := base64.StdEncoding.DecodeString(string(req.Header.Peek("Auth-Token")))
	if err != nil {
		return simply.ErrUnauthorized("unauthorized b64")
	}

	return authRequest{token}
}

func (s *Auth) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(authRequest)

	res, err := s.Cryptor.Decrypt(req.Token)
	if err != nil || len(res) != 16 {
		return simply.ErrUnauthorized("unauthorized")
	}

	uid := binary.LittleEndian.Uint64(res)

	flow.UpdateContext(context.WithValue(flow.Context(), "user_id", uid))

	return nil
}
