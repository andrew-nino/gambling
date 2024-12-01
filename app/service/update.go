package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"test_task_app/config"
	"test_task_app/helper"
)

func UpdateMatches(ctx context.Context, config config.Config, requestSemaphore chan struct{}, chanMatchesData chan map[string]interface{}, sm config.SportMode) {

	var matchesDataLock sync.Mutex
	var client *http.Client = &http.Client{}

	matchData := NewMatchData(config)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			matchData.Log.Printf("Updating %s %s matches...", sm.Sport, sm.Mode)
			err := matchData.Get(ctx, client, sm)
			if err != nil {
				matchData.Log.Printf("Error updating %s %s matches: %v", sm.Sport, sm.Mode, err)
				continue
			}

			newMatchesData := make(map[string]interface{})
			var wg sync.WaitGroup

			for _, event := range matchData.Data["events"].([]interface{}) {
				eventData := event.(map[string]interface{})["event"].(map[string]interface{})
				if strings.EqualFold(strings.ToUpper(eventData["sport"].(string)), strings.ToUpper(sm.Sport)) {
					matchID := int(eventData["id"].(float64))
					wg.Add(1)
					go func(matchID int) {
						defer wg.Done()
						result, err := matchData.Fetch(ctx, requestSemaphore, matchID, client)
						if err == nil && result != nil {

							processedData, err := helper.ProcessMatchData(result)
							if err != nil {
								matchData.Log.Printf("Error processing match data: %v", err)
								return
							}
							matchesDataLock.Lock()
							newMatchesData[strconv.Itoa(processedData.EventID)] = processedData
							matchesDataLock.Unlock()

						}
					}(matchID)
				}
			}
			wg.Wait()

			chanMatchesData <- newMatchesData

			matchData.Log.Printf("Updated %d %s %s matches", len(newMatchesData), sm.Sport, sm.Mode)

			interval := config.LiveUpdateInterval
			if sm.Mode != Live {
				interval = config.PrematchUpdateInterval
			}
			time.Sleep(interval)
		}
	}
}
