package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"test_task_app/config"
	"test_task_app/helper"

	"math/rand"

	"github.com/sirupsen/logrus"
)

const (
	Live = "Live"
)

type MatchData struct {
	cfg  config.Config
	Log  *logrus.Logger
	Data map[string]interface{}
}

func NewMatchData(cfg config.Config) *MatchData {

	logg := SetLogrus(cfg.LogLevel)
	return &MatchData{
		cfg:  cfg,
		Log:  logg,
		Data: make(map[string]interface{}),
	}
}

func (md *MatchData) Get(ctx context.Context, client *http.Client, sm config.SportMode) error {
	md.Log.WithFields(logrus.Fields{"op": "service.MatchData.Get"})
	md.Log.Infof("start getting matches for sport=%v mode=%v", sm.Sport, sm.Mode)

	now := time.Now().UTC()
	maxStartTime := now.Add(24 * time.Hour)

	var baseURL string
	var params *url.Values
	if sm.Mode == Live {
		params = setBaseParams(md.cfg)
		params.Add("useCombined", "true")
		params.Add("useCombinedLive", "true")
		baseURL = fmt.Sprintf(md.cfg.RawURLgetMatchesIsLive, md.cfg.UnibetAPIBase+md.cfg.APICountryCode, strings.ToLower(sm.Sport))

	} else {
		params = setBaseParams(md.cfg)
		params.Add("useCombined", "true")
		baseURL = fmt.Sprintf(md.cfg.RawURLgetMatches, md.cfg.UnibetAPIBase+md.cfg.APICountryCode, strings.ToLower(sm.Sport))
	}

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		md.Log.Info("error creating request")
		return err
	}

	for key, value := range getHeaders(md.cfg) {
		req.Header.Set(key, value)
	}

	client.Timeout = md.cfg.Timeout

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		md.Log.Info("error creating response")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {

		if resp.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(resp.Body)
			if err != nil {
				md.Log.Info("error read gzip")
				return err
			}
			defer reader.Close()
			err = json.NewDecoder(reader).Decode(&md.Data)
			if err != nil {
				return err
			}
		} else {
			err := json.NewDecoder(resp.Body).Decode(&md.Data)
			if err != nil {
				return err
			}
		}
		filteredEvents := []interface{}{}
		for _, event := range md.Data["events"].([]interface{}) {
			eventData := event.(map[string]interface{})["event"].(map[string]interface{})
			startTime, _ := time.Parse(time.RFC3339, eventData["start"].(string))
			homeName := strings.ToLower(eventData["homeName"].(string))
			awayName := strings.ToLower(eventData["awayName"].(string))

			if startTime.Before(maxStartTime) && !strings.Contains(homeName, "esport") && !strings.Contains(awayName, "esport") {
				filteredEvents = append(filteredEvents, event)
			}
		}

		md.Data["events"] = filteredEvents
		md.Log.Infof("finish getting matches for sport=%v mode=%v", sm.Sport, sm.Mode)
		return nil
	} else {
		return fmt.Errorf("error getting data: HTTP %d", resp.StatusCode)
	}
}

func (md *MatchData) Fetch(ctx context.Context, requestSemaphore chan struct{}, matchID int, client *http.Client) (*helper.RawData, error) {
	md.Log.WithFields(logrus.Fields{"op": "service.MatchData.Fetch"})
	md.Log.Infof("starting getting matches for matchID=%v", matchID)

	// proxyURL, err := url.Parse(getRandomProxy(md.cfg))
	// if err != nil {
	// 	md.Log.Println("Error parsing Proxy URL: ", err)
	// 	return nil, err
	// }
	transport := http.Transport{
		// Proxy: http.ProxyURL(proxyURL),
	}
	client.Transport = &transport

	params := setBaseParams(md.cfg)
	params.Add("includeParticipants", "true")
	local_url := fmt.Sprintf(md.cfg.RawURLfetchMatch, md.cfg.UnibetAPIBase+md.cfg.APICountryCode, matchID)

	req, err := http.NewRequest("GET", local_url+"?"+params.Encode(), nil)
	if err != nil {
		md.Log.Info("error creating request")
		return nil, err
	}

	for key, value := range getHeaders(md.cfg) {
		req.Header.Set(key, value)
	}

	requestSemaphore <- struct{}{}
	defer func() { <-requestSemaphore }()

	ctx, cancel := context.WithTimeout(ctx, md.cfg.Timeout)
	defer cancel()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		md.Log.Info("error creating response")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {

		var result helper.RawData

		if resp.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(resp.Body)
			if err != nil {
				md.Log.Info("error read gzip")
				return nil, err
			}
			defer reader.Close()
			err = json.NewDecoder(reader).Decode(&result)
			if err != nil {
				return nil, err
			}
		} else {
			err := json.NewDecoder(resp.Body).Decode(&result)
			if err != nil {
				return nil, err
			}
		}
		md.Log.Infof("finished getting matches for matchID=%v", matchID)
		return &result, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("STOP")
	} else {
		return nil, fmt.Errorf("error fetching data: HTTP %d", resp.StatusCode)
	}
}

func getRandomProxy(config config.Config) string {
	return config.Proxies[rand.Intn(len(config.Proxies))]
}

func getHeaders(config config.Config) map[string]string {
	return map[string]string{
		"Accept":          "application/json, text/javascript, */*; q=0.01",
		"Accept-Encoding": "gzip, deflate, br, zstd",
		"Accept-Language": "en-US;q=0.7,en;q=0.3",
		"Connection":      "keep-alive",
		"Host":            "eu-offering-api.kambicdn.com",
		"Origin":          fmt.Sprintf("https://www.unibet.%s", config.CountryCode),
		"Referer":         fmt.Sprintf("https://www.unibet.%s/", config.CountryCode),
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "cross-site",
		"User-Agent":      config.UserAgent,
	}
}

func setBaseParams(config config.Config) *url.Values {
	params := url.Values{}
	params.Add("lang", config.Lang)
	params.Add("market", config.Market)
	params.Add("client_id", config.ClientID)
	params.Add("channel_id", config.ChannelID)
	params.Add("ncid", fmt.Sprintf("%d", time.Now().Second()*1000))
	return &params
}

func SetLogrus(level string) *logrus.Logger {

	log := logrus.New()
	logrusLevel, err := logrus.ParseLevel(level)

	if err != nil {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrusLevel)
	}

	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logFile, err := os.OpenFile("./logfile.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(logFile)

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	log.SetOutput(multiWriter)

	return log
}
