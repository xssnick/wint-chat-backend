package service

import (
	"encoding/binary"
	"encoding/hex"
	"log"
	"sort"

	"github.com/xssnick/goeasy"
	"github.com/xssnick/goeasy/simply"
)

type ChatsGet struct {
	*Service
}

func (l *ChatsGet) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	uid := flow.Context().Value("user_id").(uint64)

	chats, err := l.Repo.GetChats(flow.Context(), uid, "")
	if err != nil {
		log.Println("chats get err:", uid, err)
		return simply.ErrInternal("internal error")
	}

	type chat struct {
		ID uint64 `json:"id"`
		Name string `json:"name"`
		Avatar string `json:"avatar"`
		Text string `json:"text"`
		IsMy bool `json:"is_my"`
		Unread int `json:"unread"`
		IsSame bool `json:"is_same"`
		Time int64 `json:"time"`
	}

	nids := make([]uint64, 0, len(chats))
	res := make([]chat, 0, len(chats))
	for _, c := range chats {
		bts := make([]byte, 8)
		binary.LittleEndian.PutUint64(bts, c.ID)

		enc, err := l.Cryptor.Encrypt(bts)
		if err != nil {
			log.Println("profile id enc error:", err)
			return simply.ErrInternal("internal error")
		}

		nids = append(nids, c.ID)
		res = append(res, chat{
			ID: c.ID,
			Unread: c.Unread,
			Avatar: "/images/" + hex.EncodeToString(enc) + "/0.jpg",
			Text:   c.Message,
			IsMy:   c.MessageOwner == uid,
			IsSame: c.IsSame,
			Time: c.MessageTime.Unix(),
		})
	}


	names, err := l.Repo.GetUsersNames(flow.Context(), nids)
	if err != nil {
		log.Println("names get chats err:", uid, err)
		return simply.ErrInternal("internal error")
	}

	for i := range res {
		for n := range names {
			if res[i].ID == names[n].ID {
				res[i].Name = names[n].Name
				break
			}
		}
	}
	
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID > res[j].ID
	})

	return res
}
