package service

import (
	"encoding/binary"
	"encoding/hex"
	"log"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type Profile struct {
	*Service
}

type profileResponse struct {
	Phone       string     `json:"phone"`
	Name        string     `json:"name"`
	City        string     `json:"city"`
	Country     string     `json:"country"`
	Description string     `json:"description"`
	Birth       *time.Time `json:"birth"`
	Sex         int        `json:"sex"`
	Images      []string   `json:"images"`
}

type profileRequest struct {
	ID uint64
}

func (s *Profile) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return profileRequest{
		ID: uint64(req.URI().QueryArgs().GetUintOrZero("id")),
	}
}

func (s *Profile) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(profileRequest)

	if req.ID == 0 {
		req.ID = flow.Context().Value("user_id").(uint64)
	}

	user, err := s.Repo.GetUserProfile(flow.Context(), req.ID)
	if err != nil {
		log.Println("profile get error:", err)
		return simply.ErrInternal("server error")
	}

	imgs, err := s.ImgManager.ListImages(user.ID)
	if err != nil {
		log.Println("profile get error:", err)
		return simply.ErrInternal("server error")
	}

	bts := make([]byte, 8)
	binary.LittleEndian.PutUint64(bts, user.ID)

	enc, err := s.Cryptor.Encrypt(bts)
	if err != nil {
		log.Println("profile id enc error:", err)
		return simply.ErrInternal("internal error")
	}

	for i := range imgs {
		imgs[i] = "/images/" + hex.EncodeToString(enc) + "/" + imgs[i]
	}

	return profileResponse{
		Phone:       strconv.FormatUint(user.Phone, 10),
		Name:        user.Name,
		City:        user.City,
		Country:     user.Country,
		Description: user.Description,
		Birth:       user.Birth,
		Sex:         user.Sex,
		Images:      imgs,
	}
}
