package notify

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Notifier хранит состояние последних сигналов (анти-спам) и рассылает уведомления
type Notifier struct {
	bot        *tgbotapi.BotAPI
	lastSignal map[string]string
	mu         sync.RWMutex
	getSubs    func() map[int64]bool
}

// New создаёт Notifier. getSubs вызывается при каждой рассылке для актуального списка подписчиков
func New(bot *tgbotapi.BotAPI, getSubs func() map[int64]bool) *Notifier {
	return &Notifier{
		bot:        bot,
		lastSignal: make(map[string]string),
		getSubs:    getSubs,
	}
}

// ShouldSend возвращает true, если сигнал ещё не отправлялся (или изменился тип), и запоминает его
func (n *Notifier) ShouldSend(symbol, signalType string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastSignal[symbol] == signalType {
		return false
	}
	n.lastSignal[symbol] = signalType
	return true
}

// ClearSignal сбрасывает последний сигнал по символу (например при выходе из зоны)
func (n *Notifier) ClearSignal(symbol string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSignal[symbol] = ""
}

// SendSignal отправляет уведомление подписчикам, если анти-спам разрешает.
// SHORT и LONG оформлены по-разному для быстрого визуального отличия.
func (n *Notifier) SendSignal(symbol, signalType string, rsi float64) {
	if !n.ShouldSend(symbol, signalType) {
		return
	}

	var message string
	switch signalType {
	case "SHORT":
		message = fmt.Sprintf(
			"🔴 📉 *SHORT* — перекупленность\n\n"+
				"Symbol: `%s`\n"+
				"RSI: *%.2f*",
			symbol, rsi,
		)
	case "LONG":
		message = fmt.Sprintf(
			"🟢 📈 *LONG* — перепроданность\n\n"+
				"Symbol: `%s`\n"+
				"RSI: *%.2f*",
			symbol, rsi,
		)
	default:
		message = fmt.Sprintf("🚨 %s SIGNAL\nSymbol: %s\nRSI: %.2f", signalType, symbol, rsi)
	}

	n.BroadcastMarkdown(message)
}

// Broadcast отправляет сообщение всем подписчикам (обычный текст).
func (n *Notifier) Broadcast(message string) {
	n.broadcast(message, "")
}

// broadcast отправляет сообщение подписчикам; parseMode — "Markdown" или "HTML", пустой — без разметки.
func (n *Notifier) broadcast(message, parseMode string) {
	subs := n.getSubs()
	for chatID := range subs {
		msg := tgbotapi.NewMessage(chatID, message)
		if parseMode != "" {
			msg.ParseMode = parseMode
		}
		if _, err := n.bot.Send(msg); err != nil {
			log.Printf("Не удалось отправить сообщение %d: %v", chatID, err)
		}
	}
}

// BroadcastMarkdown отправляет сообщение с разметкой Markdown (например *жирный*, `код`).
func (n *Notifier) BroadcastMarkdown(message string) {
	n.broadcast(message, "Markdown")
}
