package service

import (
	"github.com/xssnick/goeasy"
)

type Info struct {
	*Service

	Version string
}

type infoResponse struct {
	Version string
}

func (i *Info) OnProcess(flow goeasy.Flow, p interface{}) interface{} {
	return infoResponse{Version: i.Version}
}
