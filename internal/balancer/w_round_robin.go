package balancer
import (
	"net/http"
	"sync/atomic"
	"github.com/karnesanthosh/Averon/internal/server"
)

type WeightedRoundRobin struct {
	servers []*server.Server
	current uint64
	totalweight uint64
}

func NewWeightedRoundRobin(servers []*server.Server) *WeightedRoundRobin {
   var totalw uint64
   for _,s:= range servers {
	weight := s.Weight
	if weight <= 0{
		weight =1
	}
	totalw+=uint64(weight)
}
return &WeightedRoundRobin{
	servers: servers,
	totalweight: totalw,
}
	
}
	
func (s *WeightedRoundRobin) Next(req *http.Request) *server.Server{
    if s.totalweight==0{
		return nil
	}
	for i:=0;i<len(s.servers);i++{
		next:=atomic.AddUint64(&s.current,1)
		targetWeight := (next-1)%s.totalweight
		var currentsum uint64
		for _,srv:= range s.servers{
			weight := uint64(srv.Weight)
			if weight == 0{
				weight =1
			}
			currentsum +=weight
			if targetWeight<currentsum{
				if srv.IsAlive(){
					return srv
				}
				break
			}
		}

	}
	return nil
}