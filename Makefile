# Простой Makefile для RSI бота

# Имя бинарного файла
BINARY=rsi-bot

# Собрать бота
build:
	go build -o $(BINARY) .

# Запустить бота в фоне
run:
	nohup ./$(BINARY) > bot.log 2>&1 &
	echo "Бот запущен. PID: $$!"

# Остановить бота
stop:
	pkill -f $(BINARY) || true
	echo "Бот остановлен"

# Перезапустить бота
restart: stop run

# Посмотреть логи
logs:
	tail -f bot.log

# Проверить статус
status:
	pgrep -f $(BINARY) && echo "✅ Бот работает" || echo "❌ Бот не работает"

# Очистить файлы
clean:
	rm -f $(BINARY) bot.log