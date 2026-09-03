package health

import (
	"log"
	"net/http"
	"time"

	"github.com/SaisrikarVollala/nebulagate/internal/server"
)

func CheckServer(s *server.Server) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(s.URL + "/health")

	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func StartHealthChecker(servers []*server.Server, interval time.Duration) {
	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	for {
		<-ticker.C

		for _, s := range servers {
			alive := CheckServer(s)
			if alive != s.IsAlive() {
				s.SetAlive(alive)

				if alive {
					log.Printf("[HEALTH] %s is back UP", s.ID)
				} else {
					log.Printf("[HEALTH] %s is DOWN", s.ID)
				}
			}
		}
	}
}
