// Пакет main — точка входа бота: Stoch RSI на таймфрейме 1h (Bybit), уведомление при Stoch RSI = 100.
package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"grevtsevalex/crypto-bot/internal/config"
	"grevtsevalex/crypto-bot/internal/exchange"
	"grevtsevalex/crypto-bot/internal/handlers"
	"grevtsevalex/crypto-bot/internal/notify"
	"grevtsevalex/crypto-bot/internal/rsi"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const timeframe = "60" // 1 час (Bybit interval), не настраивается

var (
	bot           *tgbotapi.BotAPI
	notifier      *notify.Notifier
	subscribers   = make(map[int64]bool)
	subscribersMu sync.RWMutex
)

func getSubscribersFile() string {
	return config.Get().SubscribersFile
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

func subscribe(chatID int64) {
	subscribersMu.Lock()
	subscribers[chatID] = true
	subscribersMu.Unlock()
	saveSubscribers()
}

func unsubscribe(chatID int64) {
	subscribersMu.Lock()
	if _, exists := subscribers[chatID]; exists {
		delete(subscribers, chatID)
		saveSubscribers()
		log.Printf("Пользователь отписался: %d", chatID)
	}
	subscribersMu.Unlock()
}

func main() {
	if err := config.Load("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}
	cfg := config.Get()
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

	h := handlers.New(bot, getSubscribers, subscribe, unsubscribe)
	go h.HandleUpdates()

	for {
		log.Println("Запуск анализа рынка (1h Stoch RSI)...")

		symbols, err := exchange.DerivativePairs()
		if err != nil {
			log.Printf("Ошибка получения пар: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		loopCfg := config.Get()
		maxPer := loopCfg.MaxSignalsPerCycle
		if maxPer <= 0 {
			maxPer = 10
		}
		sentThisCycle := 0
		for _, symbol := range symbols {
			if sentThisCycle >= maxPer {
				break
			}
			if processSymbol(symbol) {
				sentThisCycle++
			}
			time.Sleep(100 * time.Millisecond)
		}

		log.Println("Анализ завершён. Следующий запуск через 1 минуту...")
		time.Sleep(1 * time.Minute)
	}
}

// processSymbol запрашивает часовые свечи, считает Stoch RSI; шлёт уведомление только при Stoch RSI ≈ 100.
// Возвращает true, если уведомление было отправлено.
func processSymbol(symbol string) bool {
	cfg := config.Get()
	limit := cfg.CandleLimit
	if limit < 50 {
		limit = 100
	}
	closes, err := exchange.Candles(symbol, timeframe, limit)
	if err != nil {
		log.Printf("Ошибка свечей %s: %v", symbol, err)
		return false
	}

	period := cfg.Period
	if period != 7 && period != 14 && period != 21 {
		period = 14
	}
	value := rsi.CalcStochRSI(closes, period, period)
	log.Printf("%s Stoch RSI(1h,%d)=%.2f", symbol, period, value)

	thr := cfg.StochRSIThreshold
	if thr <= 0 || thr > 100 {
		thr = 99.99
	}
	if value >= thr {
		notifier.SendSignal(symbol, value, period)
		return true
	}

	return false
}
