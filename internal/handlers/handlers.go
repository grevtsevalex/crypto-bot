// Package handlers содержит обработку событий от пользователя в интерфейсе Telegram-бота:
// команды, нажатия inline-кнопок, отображение меню и настроек.
package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"grevtsevalex/crypto-bot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает апдейты бота; зависимости передаются через New.
type Handler struct {
	bot            *tgbotapi.BotAPI
	getSubscribers func() map[int64]bool
	subscribe      func(chatID int64)
	unsubscribe    func(chatID int64)
	requestRestart func() // вызов перезапускает цикл анализа RSI (при смене таймфрейма/свечей)
}

// New создаёт Handler с переданными зависимостями.
// requestRestart вызывается при изменении таймфрейма или числа свечей, чтобы цикл расчёта RSI начался заново.
func New(
	bot *tgbotapi.BotAPI,
	getSubscribers func() map[int64]bool,
	subscribe, unsubscribe func(chatID int64),
	requestRestart func(),
) *Handler {
	return &Handler{
		bot:            bot,
		getSubscribers: getSubscribers,
		subscribe:      subscribe,
		unsubscribe:    unsubscribe,
		requestRestart: requestRestart,
	}
}

// HandleUpdates запускает цикл приёма апдейтов от Telegram и передаёт их в обработчики команд и callback'ов.
// Рекомендуется вызывать в отдельной горутине.
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
			case "help":
				h.showHelp(chatID)
			case "settings":
				h.showSettings(chatID)
			}
		}
	}
}

// showMainMenu отправляет главное меню: подписаться, отписаться, статус, настройки.
func (h *Handler) showMainMenu(chatID int64) {
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подписаться", "subscribe"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отписаться", "unsubscribe"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статус подписки", "status"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🤖 *Бот RSI Сигналов*\n\nВыберите действие:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = inlineKeyboard

	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки меню %d: %v", chatID, err)
	}
}

// showSettings отправляет текущие настройки и кнопки для изменения (таймфрейм, свечи, RSI, пороги).
func (h *Handler) showSettings(chatID int64) {
	cfg := config.Get()
	text := fmt.Sprintf(
		"⚙️ *Текущие настройки*\n\n"+
			"Таймфрейм: *%s* мин\n"+
			"Число свечей: *%d*\n"+
			"Период RSI: *%d*\n"+
			"Верхний порог (перекупленность): *%.0f*\n"+
			"Нижний порог (перепроданность): *%.0f*",
		cfg.Timeframe, cfg.Limit, cfg.RSIPeriod, cfg.Overbought, cfg.Oversold,
	)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📐 Таймфрейм", "menu_tf"),
			tgbotapi.NewInlineKeyboardButtonData("🕯 Свечей", "menu_limit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Период RSI", "menu_rsi"),
			tgbotapi.NewInlineKeyboardButtonData("⬆️ Верх RSI", "menu_ob"),
			tgbotapi.NewInlineKeyboardButtonData("⬇️ Низ RSI", "menu_os"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Главное меню", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb
	h.bot.Send(msg)
}

// handleCallback обрабатывает нажатие inline-кнопки: подписка, отписка, статус, настройки или применение значения настройки.
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

	case "settings":
		h.showSettings(chatID)
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		return

	case "subscribe":
		subs := h.getSubscribers()
		if _, exists := subs[chatID]; exists {
			responseText = "⚠️ Вы уже подписаны на сигналы!"
		} else {
			h.subscribe(chatID)
			responseText = "✅ Вы успешно подписались на сигналы!"
		}
		showKeyboard = true

	case "unsubscribe":
		h.unsubscribe(chatID)
		responseText = "❌ Вы отписались от сигналов. Чтобы вернуться, нажмите /start"
		showKeyboard = false

	case "status":
		subs := h.getSubscribers()
		if _, exists := subs[chatID]; exists {
			responseText = "✅ Статус: *Активен*\nВы получаете все RSI сигналы."
		} else {
			responseText = "❌ Статус: *Неактивен*\nПодпишитесь, чтобы получать сигналы."
		}
		showKeyboard = true

	default:
		if handled, msg := h.handleSettingsCallback(chatID, data); handled {
			responseText = msg
			showKeyboard = true
		} else {
			h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
			return
		}
	}

	if responseText != "" {
		msg := tgbotapi.NewMessage(chatID, responseText)
		msg.ParseMode = "Markdown"
		if showKeyboard {
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📋 Главное меню", "main_menu"),
				),
			)
		}
		h.bot.Send(msg)
	}

	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
}

// handleSettingsCallback обрабатывает callback настроек: показ подменю выбора (menu_tf и т.д.) или применение значения (tf_5, ob_85).
// Для tf и limit после обновления конфига вызывается requestRestart.
func (h *Handler) handleSettingsCallback(chatID int64, data string) (handled bool, responseText string) {
	switch data {
	case "menu_tf":
		h.sendSettingsKeyboard(chatID, "Выберите таймфрейм (минуты):", [][]string{
			{"5", "tf_5"}, {"15", "tf_15"}, {"30", "tf_30"}, {"60", "tf_60"},
			{"240", "tf_240"},
		}, "settings")
		return true, ""

	case "menu_limit":
		h.sendSettingsKeyboard(chatID, "Число свечей для расчёта:", [][]string{
			{"50", "limit_50"}, {"100", "limit_100"}, {"200", "limit_200"},
		}, "settings")
		return true, ""

	case "menu_rsi":
		h.sendSettingsKeyboard(chatID, "Период RSI:", [][]string{
			{"7", "rsi_7"}, {"14", "rsi_14"}, {"21", "rsi_21"},
		}, "settings")
		return true, ""

	case "menu_ob":
		h.sendSettingsKeyboard(chatID, "Верхний порог RSI (перекупленность):", [][]string{
			{"70", "ob_70"}, {"75", "ob_75"}, {"80", "ob_80"}, {"85", "ob_85"},
			{"90", "ob_90"}, {"95", "ob_95"}, {"100", "ob_100"},
		}, "settings")
		return true, ""

	case "menu_os":
		h.sendSettingsKeyboard(chatID, "Нижний порог RSI (перепроданность):", [][]string{
			{"0", "os_0"}, {"15", "os_15"}, {"20", "os_20"}, {"25", "os_25"}, {"30", "os_30"},
		}, "settings")
		return true, ""
	}

	parts := strings.SplitN(data, "_", 2)
	if len(parts) != 2 {
		return false, ""
	}
	key, val := parts[0], parts[1]

	switch key {
	case "tf":
		_ = config.Update(func(c *config.Config) { c.Timeframe = val })
		h.requestRestart()
		return true, fmt.Sprintf("✅ Таймфрейм: %s мин. Цикл анализа перезапущен.", val)
	case "limit":
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			_ = config.Update(func(c *config.Config) { c.Limit = v })
			h.requestRestart()
			return true, fmt.Sprintf("✅ Число свечей: %d. Цикл анализа перезапущен.", v)
		}
	case "rsi":
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			_ = config.Update(func(c *config.Config) { c.RSIPeriod = v })
			return true, fmt.Sprintf("✅ Период RSI: %d", v)
		}
	case "ob":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			_ = config.Update(func(c *config.Config) { c.Overbought = v })
			return true, fmt.Sprintf("✅ Верхний порог RSI: %.0f", v)
		}
	case "os":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			_ = config.Update(func(c *config.Config) { c.Oversold = v })
			return true, fmt.Sprintf("✅ Нижний порог RSI: %.0f", v)
		}
	}

	return false, ""
}

// sendSettingsKeyboard отправляет сообщение с кнопками выбора (options: подпись, callback_data) и кнопкой «Назад».
func (h *Handler) sendSettingsKeyboard(chatID int64, text string, options [][]string, backCallback string) {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, opt := range options {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(opt[0], opt[1]))
		if (i+1)%3 == 0 || i == len(options)-1 {
			rows = append(rows, row)
			row = nil
		}
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", backCallback)))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.bot.Send(msg)
}

// unsubscribeUser снимает пользователя с подписки (команда /stop) и отправляет подтверждение.
func (h *Handler) unsubscribeUser(chatID int64) {
	h.unsubscribe(chatID)
	msg := tgbotapi.NewMessage(chatID, "❌ Вы отписались от сигналов")
	h.bot.Send(msg)
}

// checkSubscriptionStatus отправляет пользователю статус подписки (подписан / не подписан).
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

// showHelp отправляет справку по командам и текущим параметрам расчёта RSI.
func (h *Handler) showHelp(chatID int64) {
	cfg := config.Get()
	helpText := fmt.Sprintf(`🤖 *Бот RSI Сигналов*

*Команды:*
/start - Показать главное меню
/settings - Настройки (таймфрейм, пороги RSI)
/status - Проверить статус подписки
/stop - Отписаться от сигналов
/help - Показать эту справку

*Текущие параметры:*
Таймфрейм: %s мин, свечей: %d, период RSI: %d
Сигналы: RSI > %.0f (SHORT) или RSI < %.0f (LONG)`, cfg.Timeframe, cfg.Limit, cfg.RSIPeriod, cfg.Overbought, cfg.Oversold)

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}
