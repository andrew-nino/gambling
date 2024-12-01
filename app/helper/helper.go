package helper

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Outcome struct {
	TypeName   string                   `json:"type_name"`
	Type       string                   `json:"type"`
	Line       float64                  `json:"line"`
	Odds       float64                  `json:"odds"`
	BetOfferID int                      `json:"betOfferId"`
	ID         int                      `json:"id"`
	Criterion  map[string]interface{}   `json:"criterion"`
	Path       []map[string]interface{} `json:"path"`
}

type Event struct {
	ID       int                    `json:"id"`
	HomeName string                   `json:"homeName"`
	AwayName string                   `json:"awayName"`
	Start    string                   `json:"start"`
	Sport    string                   `json:"sport"`
	Group    string                   `json:"group"`
	Path     []map[string]interface{} `json:"path"`
}

type BetOffer struct {
	Suspended bool                   `json:"suspended"`
	Outcomes  []Outcome              `json:"outcomes"`
	Criterion map[string]interface{} `json:"criterion"`
}

type RawData struct {
	Events    []Event    `json:"events"`
	BetOffers []BetOffer `json:"betOffers"`
}

type ProcessedData struct {
	EventID   int     `json:"event_id"`
	MatchName string    `json:"match_name"`
	StartTime int64     `json:"start_time"`
	HomeTeam  string    `json:"home_team"`
	AwayTeam  string    `json:"away_team"`
	Sport     string    `json:"sport"`
	League    string    `json:"league"`
	Country   string    `json:"country"`
	Outcomes  []Outcome `json:"outcomes"`
	Time      int64     `json:"time"`
	Type      string    `json:"type"`
}

func fixName(name string) string {
	splitedName := strings.SplitN(name, ",", 2)
	if len(splitedName) > 1 {
		name = strings.TrimSpace(splitedName[1] + " " + splitedName[0])
	}
	return name
}

func containsOnlyAllowedWords(label string, allowedWords []string) bool {
	words := strings.Fields(strings.ToLower(label))
	allowedSet := make(map[string]struct{}, len(allowedWords))
	for _, word := range allowedWords {
		allowedSet[word] = struct{}{}
	}
	for _, word := range words {
		if _, exists := allowedSet[word]; !exists {
			return false
		}
	}
	return true
}

func standardizeOutcome(outcome Outcome, criterion map[string]interface{}, homePlayer, awayPlayer, sport string) *string {
	label, _ := criterion["englishLabel"].(string)
	label = strings.ToLower(label)
	order, _ := criterion["order"].([]interface{})
	// line := outcome.Line
	outcomeType := outcome.Type

	var baseType string

	if sport == "Tennis" {
		if strings.Contains(label, "handicap") {
			if strings.Contains(label, "game") && len(order) == 1 && order[0] == 0.0 {
				if containsOnlyAllowedWords(label, []string{"game", "handicap"}) {
					baseType = "GAH"
				} else {
					return nil
				}
			} else if strings.Contains(label, "set") && len(order) == 1 && order[0] == 0.0 {
				if containsOnlyAllowedWords(label, []string{"set", "handicap"}) {
					baseType = "AH"
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else if strings.Contains(label, "match odds") || label == "noteringen wedstrijd" {
			if len(order) == 1 && order[0] == 0.0 && containsOnlyAllowedWords(label, []string{"match", "odds", "noteringen", "wedstrijd"}) {
				baseType = ""
			} else {
				return nil
			}
		} else if strings.Contains(label, "set") && !strings.Contains(label, "game") && !strings.Contains(label, "point") && !strings.Contains(label, "total") {
			if len(order) == 1 && order[0].(float64) >= 1 && order[0].(float64) <= 5 && containsOnlyAllowedWords(label, []string{"set", fmt.Sprintf("%d", int(order[0].(float64)))}) {
				baseType = fmt.Sprintf("%dH", int(order[0].(float64)))
			} else {
				return nil
			}
		} else if strings.Contains(label, "total") {
			if strings.Contains(label, "games") && len(order) == 1 && order[0] == 0.0 {
				baseType = "G"
			} else if strings.Contains(label, "sets") && len(order) == 1 && order[0] == 0.0 {
				baseType = ""
			} else if strings.Contains(label, "games") && strings.Contains(label, "set") && len(order) == 1 && order[0].(float64) >= 1 && order[0].(float64) <= 5 {
				baseType = fmt.Sprintf("%dHG", int(order[0].(float64)))
			} else {
				return nil
			}
		} else {
			return nil
		}

		switch outcomeType {
		case "OT_ONE", "OT_HOME":
			result := baseType + "1"
			return &result
		case "OT_TWO", "OT_AWAY":
			result := baseType + "2"
			return &result
		case "OT_OVER":
			result := baseType + "O"
			return &result
		case "OT_UNDER":
			result := baseType + "U"
			return &result
		case "OT_CROSS":
			if baseType == "" {
				result := "X"
				return &result
			}
		}
	} else if sport == "Football" {
		englishLabel, _ := outcome.Criterion["englishLabel"].(string)
		participant, _ := outcome.Criterion["participant"].(string)
		// line := outcome.Line / 1000

		if label == "full time" || label == "1x2" {
			result := outcome.Criterion["label"].(string)
			return &result
		} else if label == "first half 1x2" || label == "half time" {
			result := "1H" + outcome.Criterion["label"].(string)
			return &result
		} else if label == "2nd half 1x2" {
			result := "2H" + outcome.Criterion["label"].(string)
			return &result
		} else if strings.Contains(label, "total goals") || strings.Contains(label, "asian total") {
			var prefix string
			if strings.Contains(label, "first half") || strings.Contains(label, "1e helft") || strings.Contains(label, "1st half") {
				prefix = "1H"
			} else if strings.Contains(label, "2nd half") || strings.Contains(label, "2e helft") {
				prefix = "2H"
			}

			if strings.Contains(label, ":") {
				return nil
			}
			if strings.Contains(label, "by") || strings.Contains(label, "door") {
				if strings.Contains(strings.ToLower(label), strings.ToLower(homePlayer)) {
					teamPrefix := "H"
					if strings.Contains(strings.ToLower(englishLabel), "over") {
						result := prefix + "T" + teamPrefix + "O"
						return &result
					} else if strings.Contains(strings.ToLower(englishLabel), "under") {
						result := prefix + "T" + teamPrefix + "U"
						return &result
					}
				} else if strings.Contains(strings.ToLower(label), strings.ToLower(awayPlayer)) {
					teamPrefix := "A"
					if strings.Contains(strings.ToLower(englishLabel), "over") {
						result := prefix + "T" + teamPrefix + "O"
						return &result
					} else if strings.Contains(strings.ToLower(englishLabel), "under") {
						result := prefix + "T" + teamPrefix + "U"
						return &result
					}
				}
			} else {
				if strings.Contains(strings.ToLower(englishLabel), "over") {
					result := prefix + "O"
					return &result
				} else if strings.Contains(strings.ToLower(englishLabel), "under") {
					result := prefix + "U"
					return &result
				}
			}
		} else if strings.Contains(label, "handicap") && !strings.Contains(label, "3") {
			var prefix string
			if strings.Contains(label, "1st half") || strings.Contains(label, "1e helft") || strings.Contains(label, "first half") {
				prefix = "1H"
			} else if strings.Contains(label, "2nd half") || strings.Contains(label, "2e helft") || strings.Contains(label, "second half") {
				prefix = "2H"
			}
			if strings.ToLower(participant) == strings.ToLower(homePlayer) {
				result := prefix + "AH1"
				return &result
			} else if strings.ToLower(participant) == strings.ToLower(awayPlayer) {
				result := prefix + "AH2"
				return &result
			}
		}
	}

	return nil
}

// func ProcessMatchData(rawData RawData) (ProcessedData, error) {
func ProcessMatchData(rawData *RawData) (ProcessedData, error) {

	event := rawData.Events[0]
	homeTeam := event.HomeName
	awayTeam := event.AwayName
	startTime, err := time.Parse(time.RFC3339, event.Start)
	if err != nil {
		return ProcessedData{}, err
	}
	startTimestamp := startTime.Unix()
	currentTime := time.Now().Unix() - 60*10
	matchType := "PreMatch"
	if startTimestamp <= currentTime {
		matchType = "Live"
	}

	var processedData ProcessedData
	if strings.ToLower(event.Sport) == "tennis" {
		processedData = ProcessedData{
			EventID:   event.ID,
			MatchName: fmt.Sprintf("%s vs %s", fixName(homeTeam), fixName(awayTeam)),
			StartTime: startTimestamp,
			HomeTeam:  fixName(homeTeam),
			AwayTeam:  fixName(awayTeam),
			Sport:     strings.Title(event.Sport),
			League:    "Unknown",
			Country:   "Unknown",
			Outcomes:  []Outcome{},
			Time:      time.Now().Unix(),
			Type:      matchType,
		}
	} else {
		processedData = ProcessedData{
			EventID:   event.ID,
			MatchName: fmt.Sprintf("%s vs %s", homeTeam, awayTeam),
			StartTime: startTimestamp,
			HomeTeam:  homeTeam,
			AwayTeam:  awayTeam,
			Sport:     strings.Title(event.Sport),
			League:    event.Group,
			Country:   "Unknown",
			Outcomes:  []Outcome{},
			Time:      time.Now().Unix(),
			Type:      matchType,
		}
	}

	for _, offer := range rawData.BetOffers {
		if offer.Suspended {
			continue
		}
		for _, outcome := range offer.Outcomes {
			if outcome.Criterion["status"] == "OPEN" {
				if len(offer.Criterion) == 0 {
					continue
				}
				standardizedType := standardizeOutcome(outcome, offer.Criterion, homeTeam, awayTeam, strings.Title(event.Sport))
				if standardizedType != nil {
					processedOutcome := Outcome{
						TypeName:   offer.Criterion["englishLabel"].(string),
						Type:       *standardizedType,
						Line:       outcome.Line / 1000,
						Odds:       outcome.Odds / 1000,
						BetOfferID: outcome.BetOfferID,
						ID:         outcome.ID,
						Criterion:  offer.Criterion,
						Path:       event.Path,
					}
					processedData.Outcomes = append(processedData.Outcomes, processedOutcome)
				}
			}
		}
	}

	matchName := fmt.Sprintf("%s vs %s", processedData.HomeTeam, processedData.AwayTeam)
	matchName = strings.ReplaceAll(matchName, "/", "")

	saveOddsToJSONL(matchName, processedData)

	return processedData, nil
}

func saveOddsToJSONL(matchName string, data ProcessedData) {
	
	file, err := os.OpenFile(fmt.Sprintf("/odds_data/%s.jsonl", matchName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	dataBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling data:", err)
		return
	}

	if _, err := file.Write(dataBytes); err != nil {
		fmt.Println("Error writing to file:", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		fmt.Println("Error writing newline to file:", err)
	}
}
