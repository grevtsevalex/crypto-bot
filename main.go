// Пакет main — точка входа бота: RSI/Stoch RSI на выбранном таймфрейме Bybit,
// уведомление при достижении порога по линии Stoch RSI %K.
package main

import (
	"encoding/json"
	"flag"
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

const (
	canonicalRSIPeriod         = 14
	canonicalStochPeriod       = 14
	canonicalSmoothK           = 3
	canonicalSmoothD           = 3
	canonicalRSIUpperThreshold = 70.0
	canonicalRSILowerThreshold = 30.0
	canonicalStochUpperLevel   = 99.99
	canonicalStochLowerLevel   = 0.0
	canonicalStochLowerKSlack  = 1.0
)

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
	if err := saveSubscribers(); err != nil {
		log.Printf("Ошибка сохранения подписчиков: %v", err)
	}
}

func unsubscribe(chatID int64) {
	subscribersMu.Lock()
	_, exists := subscribers[chatID]
	if exists {
		delete(subscribers, chatID)
	}
	subscribersMu.Unlock()
	if exists {
		if err := saveSubscribers(); err != nil {
			log.Printf("Ошибка сохранения подписчиков: %v", err)
		}
		log.Printf("Пользователь отписался: %d", chatID)
	}
}

func main() {
	configPath := flag.String("config", "config.json", "path to config file")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}
	cfg := config.Get()
	if cfg.TelegramToken == "" {
		log.Fatalf("Укажите telegram_token в %s", *configPath)
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
	notifier = notify.New(botApi, cfg.SignalMode, getSubscribers)

	h := handlers.New(bot, cfg.SignalMode, getSubscribers, subscribe, unsubscribe)
	go h.HandleUpdates()

	for {
		log.Printf("Запуск анализа рынка (%s, mode=%s)...", config.Get().Timeframe, config.Get().SignalMode)

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

// processSymbol запрашивает свечи выбранного таймфрейма, считает Bybit-подобные RSI и Stoch RSI
// и шлёт уведомление при достижении порога выбранного режима.
// Возвращает true, если уведомление было отправлено.
func processSymbol(symbol string) bool {
	cfg := config.Get()
	limit := cfg.CandleLimit
	if limit < 50 {
		limit = 100
	}
	closes, err := exchange.Candles(symbol, cfg.Timeframe, limit)
	if err != nil {
		log.Printf("Ошибка свечей %s: %v", symbol, err)
		return false
	}

	values := rsi.CalcStochRSI(closes, canonicalRSIPeriod, canonicalStochPeriod, canonicalSmoothK, canonicalSmoothD)
	log.Printf(
		"%s RSI(%s,%d)=%.2f Stoch RSI(%d,%d,%d) raw=%.2f K=%.2f D=%.2f",
		symbol, cfg.Timeframe, canonicalRSIPeriod, values.RSI, canonicalStochPeriod, canonicalSmoothK, canonicalSmoothD, values.RawK, values.K, values.D,
	)

	if shouldSignal(cfg.SignalMode, values.RSI, values.RawK, values.K) {
		notifier.SendSignal(symbol, cfg.Timeframe, values.RSI, values.K, canonicalRSIPeriod, canonicalStochPeriod, canonicalSmoothK, canonicalSmoothD)
		return true
	}
	notifier.ClearSignalState(symbol)

	return false
}

func shouldSignal(signalMode string, rsiValue, rawKValue, kValue float64) bool {
	if signalMode == "lower" {
		if rsiValue > canonicalRSILowerThreshold {
			return false
		}
		// После SMA(3) линия %K часто остается чуть выше нуля даже при raw Stoch RSI = 0.
		// Для нижнего сигнала считаем касание низа валидным, если raw уже на нуле
		// или сглаженный %K визуально остается у пола.
		return rawKValue <= canonicalStochLowerLevel || kValue <= canonicalStochLowerKSlack
	}
	return rsiValue >= canonicalRSIUpperThreshold && kValue >= canonicalStochUpperLevel
}
