package balancer
import (
	"net/http"
	"github.com/karnesanthosh/Averon/internal/server"
)

type LeastConnections struct{
	servers []*server.Server

}


func NewLeastConnections(servers []*server.Server) *LeastConnections {
	return &LeastConnections{
		servers: servers,
	}

}

func (s *LeastConnections) Next(req *http.Request) *server.Server{
	var best *server.Server
	var minconns int64 = -1

	for _,srv := range s.servers{
		if !srv.IsAlive(){
			continue
		}
		conns := srv.GetConnections()
		if minconns == -1 || conns<minconns{
			minconns=conns
			best =srv
	}

}
        return best
}

