// Package notify рассылает подписчикам уведомления о верхней или нижней зоне RSI/Stoch RSI.
package notify

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Notifier struct {
	bot        *tgbotapi.BotAPI
	signalMode string
	lastSignal map[string]string
	mu         sync.RWMutex
	getSubs    func() map[int64]bool
}

func New(bot *tgbotapi.BotAPI, signalMode string, getSubs func() map[int64]bool) *Notifier {
	return &Notifier{
		bot:        bot,
		signalMode: signalMode,
		lastSignal: make(map[string]string),
		getSubs:    getSubs,
	}
}

// ShouldSend возвращает true, если для символа ещё не отправляли сигнал в текущем заходе в зону >= порога.
func (n *Notifier) ShouldSend(symbol string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastSignal[symbol] == n.signalMode {
		return false
	}
	n.lastSignal[symbol] = n.signalMode
	return true
}

// ClearSignalState сбрасывает состояние символа после выхода из активной зоны.
func (n *Notifier) ClearSignalState(symbol string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.lastSignal, symbol)
}

// SendSignal отправляет уведомление о выбранной зоне RSI/Stoch RSI, если ещё не отправляли для этого символа.
func (n *Notifier) SendSignal(symbol, timeframe string, rsiValue, kValue float64, rsiPeriod, stochPeriod, smoothK, smoothD int) {
	if !n.ShouldSend(symbol) {
		return
	}
	title := "🔴 *Upper RSI/Stoch RSI*"
	if n.signalMode == "lower" {
		title = "🟢 *Lower RSI/Stoch RSI*"
	}
	message := fmt.Sprintf("%s\n\nSymbol: `%s`\nRSI: *%.2f*\nStoch RSI %%K: *%.2f*\n\nТаймфрейм: %s\nRSI period: %d\nStoch period: %d\nSmoothing: %d/%d",
		title, symbol, rsiValue, kValue, timeframe, rsiPeriod, stochPeriod, smoothK, smoothD)
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
