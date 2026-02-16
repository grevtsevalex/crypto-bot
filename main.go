// Пакет main — точка входа Telegram-бота, который считает RSI по криптопарам (свечи с Bybit)
// и рассылает подписчикам сигналы при перекупленности (SHORT) или перепроданности (LONG).
// Конфиг и подписчики хранятся в JSON-файлах; настройки можно менять через бота.
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

var (
	bot           *tgbotapi.BotAPI
	notifier      *notify.Notifier
	subscribers   = make(map[int64]bool) // chatID -> true, доступ под subscribersMu
	subscribersMu sync.RWMutex
	restartCh     = make(chan struct{}, 1) // при отправке значения цикл анализа перезапускается
)

// getSubscribersFile возвращает путь к файлу подписчиков из текущего конфига.
func getSubscribersFile() string {
	return config.Get().SubscribersFile
}

// loadSubscribers загружает список подписчиков из JSON-файла в память.
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

// getSubscribers возвращает копию карты подписчиков (chatID) для безопасного чтения из других горутин.
func getSubscribers() map[int64]bool {
	subscribersMu.RLock()
	defer subscribersMu.RUnlock()

	out := make(map[int64]bool, len(subscribers))
	for k, v := range subscribers {
		out[k] = v
	}
	return out
}

// subscribe добавляет пользователя в подписчики и сохраняет файл.
func subscribe(chatID int64) {
	subscribersMu.Lock()
	subscribers[chatID] = true
	subscribersMu.Unlock()
	saveSubscribers()
}

// unsubscribe удаляет пользователя из подписчиков и сохраняет файл.
func unsubscribe(chatID int64) {
	subscribersMu.Lock()
	if _, exists := subscribers[chatID]; exists {
		delete(subscribers, chatID)
		saveSubscribers()
		log.Printf("Пользователь отписался: %d", chatID)
	}
	subscribersMu.Unlock()
}

// requestRestart отправляет в канал сигнал перезапуска цикла анализа (без блокировки, если канал занят).
func requestRestart() {
	select {
	case restartCh <- struct{}{}:
	default:
	}
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

	h := handlers.New(bot, getSubscribers, subscribe, unsubscribe, requestRestart)
	go h.HandleUpdates()

	for {
	restart:
		log.Println("Запуск анализа рынка...")

		symbols, err := exchange.DerivativePairs()
		if err != nil {
			log.Printf("Ошибка получения пар: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, symbol := range symbols {
			select {
			case <-restartCh:
				log.Println("Перезапуск цикла по запросу (изменён таймфрейм или число свечей)")
				goto restart
			default:
			}

			cfg := config.Get()
			processSymbol(symbol, cfg)
			time.Sleep(100 * time.Millisecond)
		}

		log.Println("Анализ завершён. Следующий запуск через 1 минуту...")
		time.Sleep(1 * time.Minute)
	}
}

// processSymbol запрашивает свечи по символу, считает RSI и при выходе за пороги отправляет сигнал подписчикам.
func processSymbol(symbol string, cfg config.Config) {
	closes, err := exchange.Candles(symbol, cfg.Timeframe, cfg.Limit)
	if err != nil {
		log.Printf("Ошибка свечей %s: %v", symbol, err)
		return
	}

	value := rsi.Calc(closes, cfg.RSIPeriod)
	log.Printf("%s RSI=%.2f", symbol, value)

	if value >= cfg.Overbought {
		notifier.SendSignal(symbol, "SHORT", value, cfg.Timeframe, cfg.Limit)
	} else if value <= cfg.Oversold {
		notifier.SendSignal(symbol, "LONG", value, cfg.Timeframe, cfg.Limit)
	} else {
		notifier.ClearSignal(symbol)
	}
}
