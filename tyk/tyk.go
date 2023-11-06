package tyk

import (
	"crypto/tls"
	"github.com/sirupsen/logrus"
	"net/http"
	"tyk/tyk/bootstrap/data"
)

type Service struct {
	httpClient http.Client
	appArgs    *data.Config
	l          *logrus.Logger
}

func NewService(args *data.Config, l *logrus.Logger) Service {
	tp := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: args.InsecureSkipVerify},
	}

	return Service{httpClient: http.Client{Transport: tp}, appArgs: args, l: l}
}
