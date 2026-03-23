// Package handlers обрабатывает команды и кнопки бота: подписка, отписка, статус, настройки, справка.
package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"grevtsevalex/crypto-bot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot            *tgbotapi.BotAPI
	getSubscribers func() map[int64]bool
	subscribe      func(chatID int64)
	unsubscribe    func(chatID int64)
}

func New(
	bot *tgbotapi.BotAPI,
	getSubscribers func() map[int64]bool,
	subscribe, unsubscribe func(chatID int64),
) *Handler {
	return &Handler{
		bot:            bot,
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
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подписаться", "subscribe"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отписаться", "unsubscribe"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статус подписки", "status"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "🤖 *Бот Stoch RSI (1h)*\n\nУведомление при достижении порога Stoch RSI.\nВыберите действие:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки меню %d: %v", chatID, err)
	}
}

func (h *Handler) showSettingsOverview(chatID int64) {
	cfg := config.Get()
	text := fmt.Sprintf(
		"⚙️ *Настройки*\n\n"+
			"Таймфрейм: *1h* (фикс)\n"+
			"Период RSI/Stoch: *%d*\n"+
			"Порог Stoch RSI: *%.2f*\n"+
			"Макс. сигналов за цикл: *%d*\n"+
			"Свечей для расчёта: *%d*\n\n"+
			"Выберите, что изменить:",
		cfg.Period, cfg.StochRSIThreshold, cfg.MaxSignalsPerCycle, cfg.CandleLimit,
	)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Период", "menu_period"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Порог RSI", "menu_thr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔢 Лимит/цикл", "menu_max"),
			tgbotapi.NewInlineKeyboardButtonData("🕯 Свечей", "menu_candles"),
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
	case "menu_period":
		h.sendSubmenu(chatID, "Период RSI и Stoch (одинаковый):", [][]string{
			{"7", "period_7"}, {"14", "period_14"}, {"21", "period_21"},
		}, "settings")
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "menu_thr":
		h.sendSubmenu(chatID, "Сигнал при Stoch RSI ≥ выбранного значения:", [][]string{
			{"99.5", "thr_995"}, {"99.9", "thr_999"}, {"99.99", "thr_9999"}, {"100", "thr_100"},
		}, "settings")
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "menu_max":
		h.sendSubmenu(chatID, "Максимум уведомлений за один проход по парам:", [][]string{
			{"5", "max_5"}, {"10", "max_10"}, {"20", "max_20"}, {"50", "max_50"},
		}, "settings")
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "menu_candles":
		h.sendSubmenu(chatID, "Число часовых свечей для расчёта:", [][]string{
			{"50", "candles_50"}, {"100", "candles_100"}, {"200", "candles_200"},
		}, "settings")
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return
	case "period_7", "period_14", "period_21":
		var p int
		switch data {
		case "period_7":
			p = 7
		case "period_14":
			p = 14
		case "period_21":
			p = 21
		}
		_ = config.Update(func(c *config.Config) { c.Period = p })
		responseText = fmt.Sprintf("✅ Период RSI/Stoch: %d", p)
		showKeyboard = true
	case "thr_995":
		_ = config.Update(func(c *config.Config) { c.StochRSIThreshold = 99.5 })
		responseText = "✅ Порог Stoch RSI: 99.5"
		showKeyboard = true
	case "thr_999":
		_ = config.Update(func(c *config.Config) { c.StochRSIThreshold = 99.9 })
		responseText = "✅ Порог Stoch RSI: 99.9"
		showKeyboard = true
	case "thr_9999":
		_ = config.Update(func(c *config.Config) { c.StochRSIThreshold = 99.99 })
		responseText = "✅ Порог Stoch RSI: 99.99"
		showKeyboard = true
	case "thr_100":
		_ = config.Update(func(c *config.Config) { c.StochRSIThreshold = 100 })
		responseText = "✅ Порог Stoch RSI: 100"
		showKeyboard = true
	case "max_5", "max_10", "max_20", "max_50":
		parts := strings.SplitN(data, "_", 2)
		if v, err := strconv.Atoi(parts[1]); err == nil {
			_ = config.Update(func(c *config.Config) { c.MaxSignalsPerCycle = v })
			responseText = fmt.Sprintf("✅ Макс. сигналов за цикл: %d", v)
			showKeyboard = true
		}
	case "candles_50", "candles_100", "candles_200":
		parts := strings.SplitN(data, "_", 2)
		if v, err := strconv.Atoi(parts[1]); err == nil {
			_ = config.Update(func(c *config.Config) { c.CandleLimit = v })
			responseText = fmt.Sprintf("✅ Свечей для расчёта: %d", v)
			showKeyboard = true
		}
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
	helpText := fmt.Sprintf(`🤖 *Бот Stoch RSI (1h)*

*Команды:*
/start — главное меню
/settings — все настройки
/status — статус подписки
/stop — отписаться
/help — эта справка

*Текущие параметры:*
Период: *%d*, порог Stoch RSI: *%.2f*
Лимит за цикл: *%d*, свечей: *%d*

Таймфрейм всегда 1h.`, cfg.Period, cfg.StochRSIThreshold, cfg.MaxSignalsPerCycle, cfg.CandleLimit)
	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}
