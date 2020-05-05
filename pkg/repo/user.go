package repo

import (
	"time"
)

type User struct {
	ID    uint64
	Phone string
	Name  string
}

type UserEdit struct {
	Phone       uint64
	Name        string
	Description string
	City        string
	Country     string
	Birth       time.Time
	Sex         int
}

type UserProfile struct {
	ID          uint64
	Phone       uint64
	Name        string
	City        string
	Country     string
	Description string
	RegFinished bool
	Birth       *time.Time
	Sex         int
	CreatedAt   time.Time
	Mode        int
}

type UserNameID struct {
	ID   uint64
	Name string
}
