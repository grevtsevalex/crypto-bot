// Package config хранит настройки бота (токен, таймфрейм и лимиты).
package config

import (
	"encoding/json"
	"os"
	"sync"
)

// Config — параметры бота.
type Config struct {
	TelegramToken      string `json:"telegram_token"`
	SubscribersFile    string `json:"subscribers_file"`
	SignalMode         string `json:"signal_mode"`
	Timeframe          string `json:"timeframe"`
	MaxSignalsPerCycle int    `json:"max_signals_per_cycle"` // макс. уведомлений за один проход по парам
	CandleLimit        int    `json:"candle_limit"`          // число часовых свечей для расчёта
}

var (
	cfg     Config
	cfgMu   sync.RWMutex
	cfgPath string
)

// Default возвращает значения по умолчанию.
func Default() Config {
	return Config{
		SubscribersFile:    "subscribers.json",
		SignalMode:         "upper",
		Timeframe:          "60",
		MaxSignalsPerCycle: 10,
		CandleLimit:        100,
	}
}

func normalize(c *Config) {
	switch c.SignalMode {
	case "upper", "lower":
	default:
		c.SignalMode = "upper"
	}
	if c.SubscribersFile == "" {
		if c.SignalMode == "lower" {
			c.SubscribersFile = "subscribers.lower.json"
		} else {
			c.SubscribersFile = "subscribers.json"
		}
	}
	switch c.Timeframe {
	case "5", "15", "60", "240", "D":
	default:
		c.Timeframe = "60"
	}
	if c.MaxSignalsPerCycle <= 0 {
		c.MaxSignalsPerCycle = 10
	}
	if c.CandleLimit < 50 || c.CandleLimit > 500 {
		c.CandleLimit = 100
	}
}

// Load загружает конфиг из файла; при отсутствии создаёт с дефолтами.
func Load(path string) error {
	cfgPath = path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = Default()
			return Save()
		}
		return err
	}
	cfgMu.Lock()
	defer cfgMu.Unlock()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	normalize(&cfg)
	return nil
}

// Save сохраняет конфиг в файл.
func Save() error {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}

// Get возвращает копию текущего конфига.
func Get() Config {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return cfg
}

// Update обновляет конфиг и сохраняет в файл.
func Update(updater func(*Config)) error {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	updater(&cfg)
	normalize(&cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}
