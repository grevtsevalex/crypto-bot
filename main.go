package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"grevtsevalex/crypto-bot/internal/exchange"
	"grevtsevalex/crypto-bot/internal/notify"
	"grevtsevalex/crypto-bot/internal/rsi"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	bot           *tgbotapi.BotAPI
	notifier      *notify.Notifier
	subscribers   = make(map[int64]bool)
	subscribersMu sync.RWMutex
)

func getSubscribersFile() string {
	return GetConfig().SubscribersFile
}

func loadSubscribers() error {
	path := getSubscribersFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	subscribersMu.Lock()
	defer subscribersMu.Unlock()

	return json.Unmarshal(data, &subscribers)
}

func saveSubscribers() error {
	subscribersMu.RLock()
	defer subscribersMu.RUnlock()

	data, err := json.MarshalIndent(subscribers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(getSubscribersFile(), data, 0644)
}

func getSubscribers() map[int64]bool {
	subscribersMu.RLock()
	defer subscribersMu.RUnlock()

	out := make(map[int64]bool, len(subscribers))
	for k, v := range subscribers {
		out[k] = v
	}
	return out
}

func main() {
	if err := LoadConfig("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	cfg := GetConfig()
	if cfg.TelegramToken == "" {
		log.Fatal("Укажите telegram_token в config.json")
	}

	if err := loadSubscribers(); err != nil {
		log.Printf("Ошибка загрузки подписчиков: %v", err)
	} else {
		log.Printf("Загружено %d подписчиков", len(subscribers))
	}

	botApi, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal("Ошибка инициализации бота:", err)
	}
	bot = botApi
	notifier = notify.New(botApi, getSubscribers)

	go handleUpdates()

	for {
		log.Println("Запуск анализа рынка...")

		symbols, err := exchange.TradingPairs()
		if err != nil {
			log.Printf("Ошибка получения пар: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, symbol := range symbols {
			cfg := GetConfig()
			processSymbol(symbol, cfg)
			time.Sleep(100 * time.Millisecond)
		}

		log.Println("Анализ завершён. Следующий запуск через 1 минуту...")
		time.Sleep(1 * time.Minute)
	}
}

func processSymbol(symbol string, cfg Config) {
	closes, err := exchange.Candles(symbol, cfg.Timeframe, cfg.Limit)
	if err != nil {
		log.Printf("Ошибка свечей %s: %v", symbol, err)
		return
	}

	value := rsi.Calc(closes, cfg.RSIPeriod)
	log.Printf("%s RSI=%.2f", symbol, value)

	if value >= cfg.Overbought {
		notifier.SendSignal(symbol, "SHORT", value)
	} else if value <= cfg.Oversold {
		notifier.SendSignal(symbol, "LONG", value)
	} else {
		notifier.ClearSignal(symbol)
	}
}
