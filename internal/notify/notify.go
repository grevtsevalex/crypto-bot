// Package notify рассылает подписчикам уведомление при верхней зоне RSI и Stoch RSI %K (1h).
package notify

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const signalType = "upper" // один тип сигнала — верхняя зона RSI/Stoch RSI

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

// SendSignal отправляет уведомление о верхней зоне RSI/Stoch RSI, если ещё не отправляли для этого символа.
func (n *Notifier) SendSignal(symbol, timeframe string, rsiValue, kValue float64, rsiPeriod, stochPeriod, smoothK, smoothD int) {
	if !n.ShouldSend(symbol) {
		return
	}
	message := fmt.Sprintf(
		"🔴 *Upper RSI/Stoch RSI*\n\nSymbol: `%s`\nRSI: *%.2f*\nStoch RSI %%K: *%.2f*\n\nТаймфрейм: %s\nRSI period: %d\nStoch period: %d\nSmoothing: %d/%d",
		symbol, rsiValue, kValue, timeframe, rsiPeriod, stochPeriod, smoothK, smoothD,
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
