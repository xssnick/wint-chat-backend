package service

import (
	"log"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
	"github.com/xssnick/wint/pkg/repo"
	// "github.com/jdeng/goheif"
)

type ProfileEdit struct {
	*Service
}

type profileEditRequest struct {
	Name    string    `json:"name"`
	City    string    `json:"city"`
	Country string    `json:"country"`
	Birth   time.Time `json:"birth"`
	Sex     int       `json:"sex"`
	Images  [][]byte  `json:"images"`
}

func (s *ProfileEdit) OnHttpRequest(flow goeasy.Flow, req *fasthttp.Request) interface{} {
	return simply.JsonPtr(new(profileEditRequest), req.Body())
}

func (s *ProfileEdit) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	req := p.(*profileEditRequest)

	uid := flow.Context().Value("user_id").(uint64)

	for i, img := range req.Images {
		log.Println(i, "img len", len(img))
		if len(img) > 0 {

			/*exif, err := goheif.ExtractExif(bytes.NewReader(img))
			if err != nil {
				log.Printf("Warning: no EXIF from: %v\n", err)
			}*/

			err := s.ImgManager.SaveImage(img, uid, strconv.Itoa(i))
			if err != nil {
				log.Println("img processing err:", err)
				return simply.ErrInternal("internal error while uploading image")
			}
		}
	}

	err := s.Repo.EditProfile(flow.Context(), uid, repo.UserEdit{
		Name:    req.Name,
		City:    req.City,
		Country: req.Country,
		Birth:   req.Birth,
		Sex:     req.Sex,
	})
	if err != nil {
		log.Println("profile edit error:", err)
		return simply.ErrInternal("server error")
	}

	return simply.Empty()
}
