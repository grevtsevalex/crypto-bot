// Package notify отвечает за рассылку уведомлений подписчикам Telegram-бота.
// Хранит состояние последнего сигнала по каждому символу (анти-спам: не слать один и тот же тип повторно).
package notify

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Notifier хранит состояние последних сигналов по символу (анти-спам) и рассылает сообщения подписчикам.
type Notifier struct {
	bot        *tgbotapi.BotAPI
	lastSignal map[string]string // symbol -> "SHORT"|"LONG", чтобы не слать один и тот же сигнал повторно
	mu         sync.RWMutex
	getSubs    func() map[int64]bool // вызывается при каждой рассылке для актуального списка
}

// New создаёт Notifier. getSubs вызывается при каждой рассылке, чтобы брать актуальный список подписчиков.
func New(bot *tgbotapi.BotAPI, getSubs func() map[int64]bool) *Notifier {
	return &Notifier{
		bot:        bot,
		lastSignal: make(map[string]string),
		getSubs:    getSubs,
	}
}

// ShouldSend возвращает true, если для этого символа ещё не отправлялся такой тип сигнала (или тип изменился), и запоминает его.
func (n *Notifier) ShouldSend(symbol, signalType string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastSignal[symbol] == signalType {
		return false
	}
	n.lastSignal[symbol] = signalType
	return true
}

// ClearSignal сбрасывает последний отправленный сигнал по символу (вызывается, когда RSI вернулся в нейтральную зону).
func (n *Notifier) ClearSignal(symbol string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSignal[symbol] = ""
}

// SendSignal отправляет уведомление подписчикам, если анти-спам разрешает.
// timeframe и limit показываются в сообщении (таймфрейм в минутах, число свечей).
// SHORT и LONG оформлены по-разному для быстрого визуального отличия.
func (n *Notifier) SendSignal(symbol, signalType string, rsi float64, timeframe string, limit int) {
	if !n.ShouldSend(symbol, signalType) {
		return
	}

	paramsLine := fmt.Sprintf("Таймфрейм: %s мин, свечей: %d", timeframe, limit)

	var message string
	switch signalType {
	case "SHORT":
		message = fmt.Sprintf(
			"🔴 📉 *SHORT* — перекупленность\n\n"+
				"Symbol: `%s`\n"+
				"RSI: *%.2f*\n"+
				"%s",
			symbol, rsi, paramsLine,
		)
	case "LONG":
		message = fmt.Sprintf(
			"🟢 📈 *LONG* — перепроданность\n\n"+
				"Symbol: `%s`\n"+
				"RSI: *%.2f*\n"+
				"%s",
			symbol, rsi, paramsLine,
		)
	default:
		message = fmt.Sprintf("🚨 %s SIGNAL\nSymbol: %s\nRSI: %.2f\n%s", signalType, symbol, rsi, paramsLine)
	}

	n.BroadcastMarkdown(message)
}

// Broadcast отправляет сообщение всем подписчикам (обычный текст).
func (n *Notifier) Broadcast(message string) {
	n.broadcast(message, "")
}

// broadcast отправляет одно сообщение всем подписчикам. parseMode — "Markdown" или "HTML", пустая строка — обычный текст.
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
