package config

import (
	"flag"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"strconv"
	"strings"
)

// Config хранит настройки приложения
type Config struct {
	APIID      int32
	APIHash    string
	Channels   []string
	ServerAddr string `yaml:"server" env-required:"true"`
	Env        string `yaml:"env" env-required:"true"`
	RedisAddr  string `yaml:"redis_addr" env-required:"true"`
	RedisDB    int    `yaml:"redis_db" env-default:"0"`
	NeuroAddr  string `yaml:"neuro_addr" env-required:"true"`
	NeuroToken string `yaml:"neuro_token" env-required:"true"`
}

// Load читает настройки из переменных окружения
func Load() (*Config, error) {
	path := fetchConfigPath()
	cfg := MustLoadPath(path)
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	apiHash := os.Getenv("TELEGRAM_API_HASH")
	channelsStr := os.Getenv("CHANNELS") // через запятую
	neuroAddr := os.Getenv("NEURO_ADDR")
	neuroToken := os.Getenv("NEURO_TOKEN") // через запятую

	if apiIDStr == "" || apiHash == "" || channelsStr == "" || neuroAddr == "" || neuroToken == "" {
		return nil, fmt.Errorf("TELEGRAM_API_ID, TELEGRAM_API_HASH, NEURO_ADDR,NEURO_TOKEN и CHANNELS должны быть заданы")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}
	apiID32 := int32(apiID)

	channels := strings.Split(channelsStr, ",")

	return &Config{
		APIID:      apiID32,
		APIHash:    apiHash,
		Channels:   channels,
		ServerAddr: cfg.ServerAddr, // <— добавили
		Env:        cfg.Env,        // <— добавили
		RedisAddr:  cfg.RedisAddr,
		RedisDB:    cfg.RedisDB,
		NeuroAddr:  neuroAddr,
		NeuroToken: neuroToken,
	}, nil
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config
	///

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}
	return res
}
