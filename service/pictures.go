package service

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type Pictures struct {
	*Service
}

type picturesRequest struct {
	UserID    uint64
	ImageName string
}

func (s *Pictures) MiddlewareChain() []goeasy.HttpMiddleware {
	return []goeasy.HttpMiddleware{}
}

func (s *Pictures) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	ownerToken := flow.Context().Value("owner").(string)
	img := flow.Context().Value("img").(string)

	tokenBytes, err := hex.DecodeString(ownerToken)
	if err != nil {
		return simply.ErrNotFound("image not found")
	}

	dec, err := s.Cryptor.Decrypt(tokenBytes)
	if err != nil || len(dec) != 8 {
		return simply.ErrNotFound("image not found")
	}

	return picturesRequest{
		UserID:    binary.LittleEndian.Uint64(dec),
		ImageName: img,
	}
}

func (s *Pictures) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(picturesRequest)

	img, err := s.ImgManager.GetImage(req.UserID, req.ImageName)
	if err != nil {
		return simply.ErrNotFound("image not found")
	}

	return img
}

func (s *Pictures) OnHttpResponse(flow goeasy.Flow, result interface{}, resp *fasthttp.Response) {
	if err, ok := result.(goeasy.Error); ok {
		resp.Header.SetContentType("text/plain")
		resp.Header.SetStatusCode(err.Code())
		return
	}

	resp.Header.SetContentType("image/jpeg")

	resp.SetBody(result.([]byte))
}
