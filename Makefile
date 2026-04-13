# Простой Makefile для RSI бота

# Имя бинарного файла
BINARY=rsi-bot
UPPER_LOG=bot.log
LOWER_LOG=bot.lower.log

# Собрать бота
build:
	go build -o $(BINARY) .

# Запустить бота в фоне
run: build
	nohup ./$(BINARY) > $(UPPER_LOG) 2>&1 &
	echo "Бот запущен. PID: $$!"

# Запустить нижнего бота в фоне
run-lower: build
	nohup ./$(BINARY) -config config.lower.json > $(LOWER_LOG) 2>&1 &
	echo "Lower бот запущен. PID: $$!"

# Остановить верхнего бота
stop-upper:
	pkill -f "^./$(BINARY)$$" || true
	echo "Upper бот остановлен"

# Остановить нижнего бота
stop-lower:
	pkill -f "^./$(BINARY) -config config.lower.json$$" || true
	echo "Lower бот остановлен"

# Остановить оба бота
stop:
	$(MAKE) stop-upper
	$(MAKE) stop-lower
	echo "Все боты остановлены"

# Перезапустить бота
restart: stop run

# Посмотреть логи
logs:
	tail -f $(UPPER_LOG)

# Посмотреть логи lower бота
logs-lower:
	tail -f $(LOWER_LOG)

# Проверить статус
status:
	pgrep -f $(BINARY) && echo "✅ Бот работает" || echo "❌ Бот не работает"

# Очистить файлы
clean:
	rm -f $(BINARY) $(UPPER_LOG) $(LOWER_LOG)