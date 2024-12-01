package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Websocket  `yaml:"websocket"`
		Unibet     `yaml:"unibet"`
		Timeout    time.Duration `yaml:"timeout_on_external_service"`
		PathToData string        `yaml:"path_to_data"`
		LogLevel   string        `yaml:"log_level"`
	}

	Websocket struct {
		Host string `yaml:"websocket_host"`
		Port int    `yaml:"websocket_port"`
	}

	Unibet struct {
		UnibetAPIBase          string        `yaml:"unibet_api_base"`
		APICountryCode         string        `yaml:"api_country_code"`
		CountryCode            string        `yaml:"country_code"`
		Lang                   string        `yaml:"lang"`
		Market                 string        `yaml:"market"`
		ClientID               string        `yaml:"client_id"`
		ChannelID              string        `yaml:"channel_id"`
		Proxies                []string      `yaml:"proxies"`
		LiveUpdateInterval     time.Duration `yaml:"live_update_interval"`
		PrematchUpdateInterval time.Duration `yaml:"prematch_update_interval"`
		MatchesPerBatch        int           `yaml:"matches_per_batch"`
		SportsToParse          []SportMode   `yaml:"sports_to_parse"`
		UserAgent              string        `yaml:"user_agent"`
		RawURLfetchMatch       string        `yaml:"raw_url_fetch_match"`
		RawURLgetMatchesIsLive string        `yaml:"raw_url_get_matches_is_live"`
		RawURLgetMatches       string        `yaml:"raw_url_get_matches"`
	}

	SportMode struct {
		Sport string `yaml:"sport"`
		Mode  string `yaml:"mode"`
	}
)

// Reads the configuration from the specified path.
func NewConfig() Config {

	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

func MustLoadPath(configPath string) Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
