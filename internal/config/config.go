// Package config хранит настройки бота (токен, период, порог Stoch RSI, лимиты).
package config

import (
	"encoding/json"
	"os"
	"sync"
)

// Config — параметры бота. Дефолты совпадают с текущими «хорошими» значениями сигналов.
type Config struct {
	TelegramToken        string  `json:"telegram_token"`
	SubscribersFile      string  `json:"subscribers_file"`
	Period               int     `json:"period"`                  // 7, 14 или 21 — RSI и Stoch
	StochRSIThreshold    float64 `json:"stoch_rsi_threshold"`     // сигнал при Stoch RSI >= этого значения
	MaxSignalsPerCycle   int     `json:"max_signals_per_cycle"`   // макс. уведомлений за один проход по парам
	CandleLimit          int     `json:"candle_limit"`            // число часовых свечей для расчёта
}

var (
	cfg     Config
	cfgMu   sync.RWMutex
	cfgPath string
)

// Default возвращает конфиг по умолчанию (как сейчас в проде).
func Default() Config {
	return Config{
		SubscribersFile:     "subscribers.json",
		Period:              14,
		StochRSIThreshold:   99.99,
		MaxSignalsPerCycle:  10,
		CandleLimit:         100,
	}
}

func normalize(c *Config) {
	if c.SubscribersFile == "" {
		c.SubscribersFile = "subscribers.json"
	}
	if c.Period != 7 && c.Period != 14 && c.Period != 21 {
		c.Period = 14
	}
	if c.StochRSIThreshold <= 0 || c.StochRSIThreshold > 100 {
		c.StochRSIThreshold = 99.99
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
