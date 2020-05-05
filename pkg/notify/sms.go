package notify

import (
	"log"
)

type SMSNotifier struct {
}

func (S SMSNotifier) Notify(phone uint64, code string) error {
	log.Println("NOTIFY:", phone, code)
	return nil
}

func NewSMSNotifier() *SMSNotifier {
	return &SMSNotifier{}
}
