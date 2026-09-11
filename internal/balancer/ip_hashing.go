package balancer
import (
	"hash/fnv"
	"net/http"
	"strings"
	"github.com/karnesanthosh/Averon/internal/server"
)

type IPHash struct {
	servers []*server.Server
}

func NewIPHash(servers []*server.Server) *IPHash {
	return &IPHash{
		servers: servers,
	}
}

func (s *IPHash) Next(req *http.Request) *server.Server{
    if len(s.servers)==0{
		return nil
	}
	 ip:= req.RemoteAddr
	 if idx:= strings.LastIndex(ip,":");idx!=-1{
		ip=ip[:idx]
	 }

h:=fnv.New32a()

h.Write([]byte(ip))
hashVal:=h.Sum32()

idx:=hashVal % uint32(len(s.servers))

for i :=0;i<len(s.servers);i++{
   srv := s.servers[(int(idx)+i)%len(s.servers)]
   if srv.IsAlive(){
	return srv
   }
}
return nil
	

}