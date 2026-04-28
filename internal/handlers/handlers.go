// Package handlers обрабатывает команды и кнопки бота: подписка, отписка, статус, настройки, справка.
package handlers

import (
	"fmt"
	"log"
	"strings"

	"grevtsevalex/crypto-bot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot            *tgbotapi.BotAPI
	signalMode     string
	getSubscribers func() map[int64]bool
	subscribe      func(chatID int64)
	unsubscribe    func(chatID int64)
}

func New(
	bot *tgbotapi.BotAPI,
	signalMode string,
	getSubscribers func() map[int64]bool,
	subscribe, unsubscribe func(chatID int64),
) *Handler {
	return &Handler{
		bot:            bot,
		signalMode:     signalMode,
		getSubscribers: getSubscribers,
		subscribe:      subscribe,
		unsubscribe:    unsubscribe,
	}
}

func (h *Handler) HandleUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := h.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			h.handleCallback(update.CallbackQuery)
			continue
		}
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				h.showMainMenu(chatID)
			case "stop":
				h.unsubscribeUser(chatID)
			case "status":
				h.checkSubscriptionStatus(chatID)
			case "settings":
				h.showSettingsOverview(chatID)
			case "help":
				h.showHelp(chatID)
			}
		}
	}
}

func (h *Handler) showMainMenu(chatID int64) {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подписаться", "subscribe"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отписаться", "unsubscribe"),
		),
	}
	if config.Get().LockTimeframe {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статус подписки", "status"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статус подписки", "status"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		))
	}
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🤖 *%s*\n\n%s\nВыберите действие:", h.botTitle(), h.botDescription()))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки меню %d: %v", chatID, err)
	}
}

func (h *Handler) showSettingsOverview(chatID int64) {
	cfg := config.Get()
	if cfg.LockTimeframe {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚙️ Таймфрейм зафиксирован: *%s*", humanTimeframe(cfg.Timeframe)))
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)
		return
	}
	text := fmt.Sprintf(
		"⚙️ *Настройки*\n\n"+
			"Таймфрейм: *%s*\n"+
			"Выберите, что изменить:",
		cfg.Timeframe,
	)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🕯 Таймфрейм", "menu_timeframe"),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📋 Главное меню", "main_menu")),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb
	h.bot.Send(msg)
}

func (h *Handler) sendSubmenu(chatID int64, title string, options [][]string, back string) {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, opt := range options {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(opt[0], opt[1]))
		if (i+1)%3 == 0 || i == len(options)-1 {
			rows = append(rows, row)
			row = nil
		}
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", back)))
	msg := tgbotapi.NewMessage(chatID, title)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.bot.Send(msg)
}

func (h *Handler) handleCallback(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data

	var responseText string
	var showKeyboard bool

	switch data {
	case "main_menu":
		h.showMainMenu(chatID)
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "subscribe":
		subs := h.getSubscribers()
		if _, exists := subs[chatID]; exists {
			responseText = "⚠️ Вы уже подписаны на сигналы!"
		} else {
			h.subscribe(chatID)
			responseText = "✅ Вы подписаны на уведомления."
		}
		showKeyboard = true
	case "unsubscribe":
		h.unsubscribe(chatID)
		responseText = "❌ Вы отписались от сигналов."
		showKeyboard = true
	case "status":
		subs := h.getSubscribers()
		if _, exists := subs[chatID]; exists {
			responseText = "✅ Подписан."
		} else {
			responseText = "❌ Не подписан."
		}
		showKeyboard = true
	case "settings":
		h.showSettingsOverview(chatID)
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "menu_timeframe":
		if config.Get().LockTimeframe {
			responseText = "⚠️ Таймфрейм зафиксирован в конфиге бота."
			showKeyboard = true
			break
		}
		h.sendSubmenu(chatID, "Таймфрейм свечей:", [][]string{
			{"5m", "timeframe_5"}, {"15m", "timeframe_15"}, {"1h", "timeframe_60"}, {"4h", "timeframe_240"}, {"1D", "timeframe_D"},
		}, "settings")
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "timeframe_5", "timeframe_15", "timeframe_60", "timeframe_240", "timeframe_D":
		if config.Get().LockTimeframe {
			responseText = "⚠️ Таймфрейм зафиксирован в конфиге бота."
			showKeyboard = true
			break
		}
		value := strings.TrimPrefix(data, "timeframe_")
		_ = config.Update(func(c *config.Config) { c.Timeframe = value })
		responseText = fmt.Sprintf("✅ Таймфрейм: %s", humanTimeframe(value))
		showKeyboard = true
	default:
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	}

	if responseText != "" {
		msg := tgbotapi.NewMessage(chatID, responseText)
		msg.ParseMode = "Markdown"
		if showKeyboard {
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📋 Главное меню", "main_menu")),
			)
		}
		h.bot.Send(msg)
	}
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
}

func (h *Handler) unsubscribeUser(chatID int64) {
	h.unsubscribe(chatID)
	h.bot.Send(tgbotapi.NewMessage(chatID, "❌ Вы отписались от сигналов"))
}

func (h *Handler) checkSubscriptionStatus(chatID int64) {
	subs := h.getSubscribers()
	status := "❌ Не подписан"
	if _, exists := subs[chatID]; exists {
		status = "✅ Подписан"
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📊 *Статус подписки*\n\n%s", status))
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}

func (h *Handler) showHelp(chatID int64) {
	cfg := config.Get()
	helpText := fmt.Sprintf(`🤖 *%s*

*Команды:*
/start — главное меню
/settings — настройки
/status — статус подписки
/stop — отписаться
/help — эта справка

*Текущие параметры:*
Таймфрейм: *%s*

Расчёт индикаторов зафиксирован на канонических значениях Bybit/TradingView. %s`,
		h.botTitle(), humanTimeframe(cfg.Timeframe), h.botDescription())
	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}

func (h *Handler) botTitle() string {
	tf := humanTimeframe(config.Get().Timeframe)
	if h.signalMode == "lower" {
		return fmt.Sprintf("Бот Lower RSI/Stoch RSI %s", tf)
	}
	return fmt.Sprintf("Бот Upper RSI/Stoch RSI %s", tf)
}

func (h *Handler) botDescription() string {
	tf := humanTimeframe(config.Get().Timeframe)
	if h.signalMode == "lower" {
		return fmt.Sprintf("Уведомление только по нижней зоне RSI и Stoch RSI (%%K около 0). Таймфрейм: %s.", tf)
	}
	return fmt.Sprintf("Уведомление только по верхней зоне RSI и Stoch RSI. Таймфрейм: %s.", tf)
}

func humanTimeframe(value string) string {
	switch value {
	case "5":
		return "5m"
	case "15":
		return "15m"
	case "60":
		return "1h"
	case "240":
		return "4h"
	case "D":
		return "1D"
	default:
		return value
	}
}
