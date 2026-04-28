# Простой Makefile для RSI ботов

BINARY=rsi-bot

UPPER_60_CFG=config.upper.60.json
UPPER_240_CFG=config.upper.240.json
UPPER_D_CFG=config.upper.D.json
LOWER_60_CFG=config.lower.60.json

build:
	go build -o $(BINARY) .

run-upper-60: build
	nohup ./$(BINARY) -config $(UPPER_60_CFG) > /dev/null 2>&1 &
	echo "Upper 1h бот запущен"

run-upper-240: build
	nohup ./$(BINARY) -config $(UPPER_240_CFG) > /dev/null 2>&1 &
	echo "Upper 4h бот запущен"

run-upper-d: build
	nohup ./$(BINARY) -config $(UPPER_D_CFG) > /dev/null 2>&1 &
	echo "Upper 1D бот запущен"

run-lower-60: build
	nohup ./$(BINARY) -config $(LOWER_60_CFG) > /dev/null 2>&1 &
	echo "Lower 1h бот запущен"

run-all: build
	nohup ./$(BINARY) -config $(UPPER_60_CFG) > /dev/null 2>&1 &
	nohup ./$(BINARY) -config $(UPPER_240_CFG) > /dev/null 2>&1 &
	nohup ./$(BINARY) -config $(UPPER_D_CFG) > /dev/null 2>&1 &
	nohup ./$(BINARY) -config $(LOWER_60_CFG) > /dev/null 2>&1 &
	echo "Все 4 бота запущены"

run: run-upper-60
run-lower: run-lower-60

stop-upper-60:
	pkill -f "^./$(BINARY) -config $(UPPER_60_CFG)$$" || true
	echo "Upper 1h бот остановлен"

stop-upper-240:
	pkill -f "^./$(BINARY) -config $(UPPER_240_CFG)$$" || true
	echo "Upper 4h бот остановлен"

stop-upper-d:
	pkill -f "^./$(BINARY) -config $(UPPER_D_CFG)$$" || true
	echo "Upper 1D бот остановлен"

stop-lower-60:
	pkill -f "^./$(BINARY) -config $(LOWER_60_CFG)$$" || true
	echo "Lower 1h бот остановлен"

stop-upper: stop-upper-60
stop-lower: stop-lower-60

stop:
	$(MAKE) stop-upper-60
	$(MAKE) stop-upper-240
	$(MAKE) stop-upper-d
	$(MAKE) stop-lower-60
	echo "Все 4 бота остановлены"

restart: stop run-all

logs-upper-60:
	@echo "Логи отключены"

logs-upper-240:
	@echo "Логи отключены"

logs-upper-d:
	@echo "Логи отключены"

logs-lower-60:
	@echo "Логи отключены"

logs: logs-upper-60
logs-lower: logs-lower-60

status:
	pgrep -f $(BINARY) && echo "✅ Бот работает" || echo "❌ Бот не работает"

clean:
	rm -f $(BINARY) bot.upper.60.log bot.upper.240.log bot.upper.D.log bot.lower.60.log bot.log bot.lower.log
