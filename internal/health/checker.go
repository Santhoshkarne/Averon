package health

import (
	
	"net/http"
	"time"

	"github.com/karnesanthosh/Averon/internal/server"
	"github.com/karnesanthosh/Averon/internal/logger"
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
					logger.Global.Info("backend health changed", map[string]interface{}{
						"backend": s.ID,
						"status":  "UP",
					})
				} else {
					logger.Global.Info("backend health changed", map[string]interface{}{
						"backend": s.ID,
						"status":  "DOWN",
					})
				}
			}
		}
	}
}
