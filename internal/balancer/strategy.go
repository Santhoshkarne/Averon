package balancer

import (
	"net/http"
	"github.com/karnesanthosh/Averon/internal/server"
)

type Strategy interface {
	Next(req *http.Request) *server.Server
}

