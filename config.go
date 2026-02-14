package main

import (
	"encoding/json"
	"os"
	"sync"
)

// Config — параметры расчёта RSI и оповещений
type Config struct {
	Timeframe       string  `json:"timeframe"`        // "1", "5", "15", "60" (минуты)
	Limit           int     `json:"limit"`            // число свечей для расчёта
	RSIPeriod       int     `json:"rsi_period"`       // период RSI (обычно 14)
	Overbought      float64 `json:"overbought"`       // верхний порог RSI (сигнал SHORT)
	Oversold        float64 `json:"oversold"`         // нижний порог RSI (сигнал LONG)
	TelegramToken   string  `json:"telegram_token"`   // токен бота (не показывать в /settings)
	SubscribersFile string  `json:"subscribers_file"` // файл подписчиков
}

var (
	config      Config
	configMutex sync.RWMutex
	configPath  = "config.json"
)

// DefaultConfig возвращает конфиг по умолчанию
func DefaultConfig() Config {
	return Config{
		Timeframe:       "5",
		Limit:           100,
		RSIPeriod:       14,
		Overbought:      80.0,
		Oversold:        20.0,
		SubscribersFile: "subscribers.json",
	}
}

// LoadConfig загружает конфиг из файла или создаёт с дефолтами
func LoadConfig(path string) error {
	configPath = path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			config = DefaultConfig()
			return SaveConfig()
		}
		return err
	}
	configMutex.Lock()
	defer configMutex.Unlock()
	return json.Unmarshal(data, &config)
}

// SaveConfig сохраняет конфиг в файл
func SaveConfig() error {
	configMutex.RLock()
	defer configMutex.RUnlock()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// GetConfig возвращает копию текущего конфига (для чтения)
func GetConfig() Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return config
}

// UpdateConfig обновляет часть конфига и сохраняет (для настроек из бота)
func UpdateConfig(updater func(*Config)) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	updater(&config)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
