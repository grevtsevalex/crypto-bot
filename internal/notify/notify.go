// Package notify рассылает подписчикам уведомление при Stoch RSI = 100 (1h).
package notify

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const signalType = "100" // один тип сигнала — Stoch RSI = 100

type Notifier struct {
	bot        *tgbotapi.BotAPI
	lastSignal map[string]string
	mu         sync.RWMutex
	getSubs    func() map[int64]bool
}

func New(bot *tgbotapi.BotAPI, getSubs func() map[int64]bool) *Notifier {
	return &Notifier{
		bot:        bot,
		lastSignal: make(map[string]string),
		getSubs:    getSubs,
	}
}

// ShouldSend возвращает true, если для символа ещё не отправляли сигнал в текущем заходе в зону >= порога.
func (n *Notifier) ShouldSend(symbol string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastSignal[symbol] == signalType {
		return false
	}
	n.lastSignal[symbol] = signalType
	return true
}

// ClearSignal сбрасывает состояние символа, когда значение ушло ниже порога.
// Это позволяет отправить сигнал снова при следующем новом заходе к 100.
func (n *Notifier) ClearSignal(symbol string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSignal[symbol] = ""
}

// SendSignal отправляет уведомление «Stoch RSI (1h) = 100» по символу, если ещё не отправляли для этого символа.
// period — выбранный период RSI/Stoch (7, 14 или 21).
func (n *Notifier) SendSignal(symbol string, value float64, period int) {
	if !n.ShouldSend(symbol) {
		return
	}
	message := fmt.Sprintf(
		"🔴 *Stoch RSI (1h) = 100*\n\nSymbol: `%s`\nStoch RSI: *%.2f*\n\nТаймфрейм: 1h, период: %d",
		symbol, value, period,
	)
	n.broadcast(message, "Markdown")
}

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
