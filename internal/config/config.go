// Package config хранит и загружает настройки расчёта RSI и оповещений (таймфрейм, пороги, токен и т.д.).
package config

import (
	"encoding/json"
	"os"
	"sync"
)

// Config — параметры расчёта RSI и оповещений.
// Поля совпадают с JSON в config.json.
type Config struct {
	Timeframe       string  `json:"timeframe"`        // интервал свечей: "5", "15", "60" и т.д. (минуты)
	Limit           int     `json:"limit"`            // число свечей для расчёта RSI
	RSIPeriod       int     `json:"rsi_period"`       // период RSI (обычно 14)
	Overbought      float64 `json:"overbought"`        // верхний порог: RSI >= Overbought → сигнал SHORT
	Oversold        float64 `json:"oversold"`         // нижний порог: RSI <= Oversold → сигнал LONG
	TelegramToken   string  `json:"telegram_token"`   // токен Telegram-бота
	SubscribersFile string  `json:"subscribers_file"`  // путь к JSON с подписчиками (chatID)
}

var (
	cfg     Config   // текущий конфиг в памяти
	cfgMu   sync.RWMutex
	cfgPath string   // путь к config.json, задаётся в Load
)

// Default возвращает конфиг по умолчанию.
func Default() Config {
	return Config{
		Timeframe:       "5",
		Limit:           100,
		RSIPeriod:       14,
		Overbought:      80.0,
		Oversold:        20.0,
		SubscribersFile: "subscribers.json",
	}
}

// Load загружает конфиг из файла; при отсутствии файла создаёт его с дефолтами.
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
	return json.Unmarshal(data, &cfg)
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

// Update обновляет конфиг через функцию и сохраняет в файл.
func Update(updater func(*Config)) error {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	updater(&cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}
