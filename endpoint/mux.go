package main

import (
	"github.com/inconshreveable/muxado"
)

func GetMuxadoSession() (muxado.Session, error) {
	rwc, err := RemoteConnect()
	if err != nil {
		return nil, err
	}

	return muxado.Server(rwc, nil), nil
}
