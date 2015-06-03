package services

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func statusChecker(sl *List, secondsBetweenChecking time.Duration) {
	ticker := time.NewTicker(secondsBetweenChecking * time.Second)
	for {
		select {
		case <-ticker.C:
			sl.checkState()
		}
	}
}

func (sl *List) checkState() {
	for i := 0; i < len(sl.list); i++ {
		if sl.list[i].failedConnections <= sl.failedConnectionsLimit {
			resp, err := http.Get(fmt.Sprintf("%v://%v%v", sl.scheme, sl.list[i].address, sl.statusPath))

			if err != nil || resp.StatusCode != 200 { // TODO
				sl.updateFailedConnections(i, sl.list[i].failedConnections+1)
			} else {
				resp.Body.Close()
				sl.updateFailedConnections(i, 0)
			}
		}

		if sl.list[i].failedConnections > sl.failedConnectionsLimit {
			log.Printf("%s: removed %s\n\n", sl.name, sl.list[i].address)
			sl.removeService(i)
			i--
			continue
		}

		if sl.list[i].failedConnections != 0 {
			log.Printf("%s status check: [%s] failed %v times\n", sl.name, sl.list[i].address, sl.list[i].failedConnections)
		}
	}
}
