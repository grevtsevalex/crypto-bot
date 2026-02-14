package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	timeframe        = "5"
	limit            = 100
	rsiPeriod        = 14
	overbought       = 80.0
	oversold         = 20.0
	telegramToken    = "8296325515:AAEG-u5Ks-MlJOFMEWzG9dPzDKC1FrTDZpI"
	bot              *tgbotapi.BotAPI
	subscribers      = make(map[int64]bool)
	subscribersMutex sync.RWMutex
	subscribersFile  = "subscribers.json"
)

// анти-спам состояние
var lastSignal = make(map[string]string)

type ExchangeInfo struct {
	Symbols []struct {
		Symbol     string `json:"symbol"`
		Status     string `json:"status"`
		QuoteAsset string `json:"quoteAsset"`
	} `json:"symbols"`
}

// loadSubscribers загружает подписчиков из файла
func loadSubscribers() error {
	data, err := os.ReadFile(subscribersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	return json.Unmarshal(data, &subscribers)
}

// saveSubscribers сохраняет подписчиков в файл
func saveSubscribers() error {
	subscribersMutex.RLock()
	defer subscribersMutex.RUnlock()

	data, err := json.MarshalIndent(subscribers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(subscribersFile, data, 0644)
}

// removeSubscriber удаляет подписчика (НОВАЯ ФУНКЦИЯ)
func removeSubscriber(chatID int64) error {
	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	// Проверяем, был ли пользователь подписан
	if _, exists := subscribers[chatID]; exists {
		delete(subscribers, chatID)
		// Сохраняем обновлённый список
		if err := saveSubscribers(); err != nil {
			return err
		}
		log.Printf("Пользователь %d отписался", chatID)
	}
	return nil
}

func main() {
	if err := loadSubscribers(); err != nil {
		log.Printf("Ошибка загрузки подписчиков: %v", err)
	} else {
		log.Printf("Загружено %d подписчиков из файла", len(subscribers))
	}

	botApi, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("Ошибка инициализации бота:", err)
	}
	bot = botApi

	go handleUpdates()

	// Основной цикл для анализа RSI
	for {
		log.Println("Запуск анализа рынка...")

		// Получаем все пары
		symbols, err := getAllUSDTTradingPairs()
		if err != nil {
			log.Printf("Ошибка получения пар: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Анализируем каждую пару
		for _, symbol := range symbols {
			processSymbol(symbol)
			time.Sleep(100 * time.Millisecond) // небольшая пауза между запросами
		}

		log.Println("Анализ завершён. Следующий запуск через 15 минут...")
		time.Sleep(15 * time.Minute) // Ждём 15 минут
	}
}

func getAllUSDTTradingPairs() ([]string, error) {
	url := "https://api.binance.com/api/v3/exchangeInfo"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data ExchangeInfo
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var result []string

	for _, s := range data.Symbols {
		if s.Status == "TRADING" && s.QuoteAsset == "USDT" {
			result = append(result, s.Symbol)
		}
	}

	return result, nil
}

func processSymbol(symbol string) {
	closes, err := fetchCloses(symbol)
	if err != nil {
		fmt.Println("error fetching candles:", err)
		return
	}

	rsi := RSI(closes, rsiPeriod)

	// fmt.Printf("%s RSI=%.2f\n", symbol, rsi)

	if rsi == 0 {
		return
	}

	if rsi > overbought {
		handleSignal(symbol, "SHORT", rsi)
		// } else if rsi < oversold {
		// 	handleSignal(symbol, "LONG", rsi)
	} else {
		lastSignal[symbol] = ""
	}
}

func handleSignal(symbol, signalType string, rsi float64) {
	if lastSignal[symbol] == signalType {
		return
	}

	message := fmt.Sprintf(
		"🚨 %s SIGNAL\nSymbol: %s\nRSI: %.2f",
		signalType,
		symbol,
		rsi,
	)

	broadcast(bot, message)
	lastSignal[symbol] = signalType
}

func fetchCloses(symbol string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf(
		"https://api.bybit.com/v5/market/kline?category=linear&symbol=%s&interval=%s&limit=%d",
		symbol,
		timeframe,
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Result struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var closes []float64

	for i := len(data.Result.List) - 1; i >= 0; i-- {
		closePrice, err := strconv.ParseFloat(data.Result.List[i][4], 64)
		if err != nil {
			continue
		}
		closes = append(closes, closePrice)
	}

	return closes, nil
}

func RSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}

	var gain, loss float64

	for i := 1; i <= period; i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}

	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

type Update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

func handleUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// Обработка нажатий на inline-кнопки
		if update.CallbackQuery != nil {
			handleCallback(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		// Проверяем команды
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				showMainMenu(chatID)
			case "stop":
				unsubscribeUser(chatID)
			case "status":
				checkSubscriptionStatus(chatID)
			case "help":
				showHelp(chatID)
			}
		}
	}
}

// Показывает главное меню с inline-кнопками
func showMainMenu(chatID int64) {
	// Создаём inline-клавиатуру
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		// Первый ряд кнопок
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подписаться", "subscribe"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отписаться", "unsubscribe"),
		),
		// Второй ряд кнопок
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статус подписки", "status"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🤖 *Бот RSI Сигналов*\n\nВыберите действие:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = inlineKeyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки меню %d: %v", chatID, err)
	}
}

// Обработка нажатий на inline-кнопки
func handleCallback(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data // данные, которые мы указали при создании кнопки

	var responseText string
	var showKeyboard bool

	switch data {
	case "subscribe":
		// Проверяем, не подписан ли уже
		subscribersMutex.RLock()
		_, exists := subscribers[chatID]
		subscribersMutex.RUnlock()

		if exists {
			responseText = "⚠️ Вы уже подписаны на сигналы!"
		} else {
			// Подписываем
			subscribersMutex.Lock()
			subscribers[chatID] = true
			subscribersMutex.Unlock()
			saveSubscribers()
			responseText = "✅ Вы успешно подписались на сигналы!"
		}
		showKeyboard = true

	case "unsubscribe":
		// Отписываем
		subscribersMutex.Lock()
		delete(subscribers, chatID)
		subscribersMutex.Unlock()
		saveSubscribers()
		responseText = "❌ Вы отписались от сигналов. Чтобы вернуться, нажмите /start"
		showKeyboard = false

	case "status":
		subscribersMutex.RLock()
		_, exists := subscribers[chatID]
		subscribersMutex.RUnlock()

		if exists {
			responseText = "✅ Статус: *Активен*\nВы получаете все RSI сигналы."
		} else {
			responseText = "❌ Статус: *Неактивен*\nПодпишитесь, чтобы получать сигналы."
		}
		showKeyboard = true
	}

	// Отправляем ответ на нажатие кнопки
	msg := tgbotapi.NewMessage(chatID, responseText)
	msg.ParseMode = "Markdown"

	if showKeyboard {
		// Показываем главное меню снова
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Главное меню", "main_menu"),
			),
		)
	}

	bot.Send(msg)

	// Отвечаем на callback (убираем "часики" у кнопки)
	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Request(callback)
}

// Функция отписки
func unsubscribeUser(chatID int64) {
	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	if _, exists := subscribers[chatID]; exists {
		delete(subscribers, chatID)
		saveSubscribers()
		log.Printf("Пользователь отписался: %d", chatID)
	}

	// Отправляем подтверждение
	msg := tgbotapi.NewMessage(chatID, "❌ Вы отписались от сигналов")
	bot.Send(msg)
}

// Проверка статуса
func checkSubscriptionStatus(chatID int64) {
	subscribersMutex.RLock()
	_, exists := subscribers[chatID]
	subscribersMutex.RUnlock()

	status := "❌ Не подписан"
	if exists {
		status = "✅ Подписан"
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📊 *Статус подписки*\n\n%s", status))
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Справка
func showHelp(chatID int64) {
	helpText := `🤖 *Бот RSI Сигналов*

	*Команды:*
	/start - Показать главное меню
	/status - Проверить статус подписки
	/stop - Отписаться от сигналов
	/help - Показать эту справку

	*О боте:*
	Бот анализирует RSI на 5-минутных свечах и отправляет сигналы при достижении экстремальных значений (>80 или <20).`

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func broadcast(bot *tgbotapi.BotAPI, message string) {
	subscribersMutex.RLock()
	defer subscribersMutex.RUnlock()

	for chatID := range subscribers {
		msg := tgbotapi.NewMessage(chatID, message)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Не удалось отправить сообщение %d: %v", chatID, err)
		}
	}
}
