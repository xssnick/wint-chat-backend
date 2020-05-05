package otp

import (
	"bytes"
	"crypto/hmac"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"log"
	"math"
	"strconv"
	"time"
)

type TOTP struct {
	master []byte
	hasher func() hash.Hash
}

func (t *TOTP) Generate(base string, digs int) (long []byte, short string) {
	h := hmac.New(t.hasher, t.master)
	h.Write([]byte(base + ";;" + strconv.FormatInt(time.Now().Unix()/45, 16)))

	log.Println("gen key", base+";;"+strconv.FormatInt(time.Now().Unix()/45, 16))

	long = h.Sum(nil)
	short = fmt.Sprintf("%0*d", digs, binary.LittleEndian.Uint32(long[4:])%uint32(math.Pow10(digs)))

	return long[4:], short
}

func (t *TOTP) Validate(base string, digs int, slong []byte, sshort string) bool {
	var ok bool
	var long []byte
	for i := 0; i < 3; i++ {
		h := hmac.New(t.hasher, t.master)

		log.Println("key", i, base+";;"+strconv.FormatInt((time.Now().Unix()/45)-int64(i), 16))
		h.Write([]byte(base + ";;" + strconv.FormatInt((time.Now().Unix()/45)-int64(i), 16)))

		long = h.Sum(nil)

		log.Println(hex.EncodeToString(long[4:]), hex.EncodeToString(slong))
		if bytes.Equal(long[4:], slong) {
			ok = true
			break
		}
	}

	if !ok {
		return false
	}

	return fmt.Sprintf("%0*d", digs, binary.LittleEndian.Uint32(long[4:])%uint32(math.Pow10(digs))) == sshort
}

func NewTOTP(hasher func() hash.Hash, master []byte) *TOTP {
	return &TOTP{master: master, hasher: hasher}
}
