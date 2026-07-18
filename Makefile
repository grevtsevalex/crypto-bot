# Простой Makefile для RSI ботов

BINARY=rsi-bot

UPPER_60_CFG=config.upper.60.json
UPPER_15_CFG=config.upper.15.json
UPPER_240_CFG=config.upper.240.json
UPPER_D_CFG=config.upper.D.json
LOWER_240_CFG=config.lower.240.json
LOWER_D_CFG=config.lower.D.json

build:
	go build -o $(BINARY) .

run-upper-60: build
	nohup ./$(BINARY) -config $(UPPER_60_CFG) >> bot.upper.60.log 2>&1 &
	echo "Upper 1h бот запущен"

run-upper-15: build
	nohup ./$(BINARY) -config $(UPPER_15_CFG) >> bot.upper.15.log 2>&1 &
	echo "Upper 15m бот запущен"

run-upper-240: build
	nohup ./$(BINARY) -config $(UPPER_240_CFG) >> bot.upper.240.log 2>&1 &
	echo "Upper 4h бот запущен"

run-upper-d: build
	nohup ./$(BINARY) -config $(UPPER_D_CFG) >> bot.upper.D.log 2>&1 &
	echo "Upper 1D бот запущен"

run-lower-240: build
	nohup ./$(BINARY) -config $(LOWER_240_CFG) >> bot.lower.240.log 2>&1 &
	echo "Lower 4h бот запущен"

run-lower-d: build
	nohup ./$(BINARY) -config $(LOWER_D_CFG) >> bot.lower.D.log 2>&1 &
	echo "Lower 1D бот запущен"

run-all: build
	nohup ./$(BINARY) -config $(UPPER_60_CFG) >> bot.upper.60.log 2>&1 &
	nohup ./$(BINARY) -config $(UPPER_15_CFG) >> bot.upper.15.log 2>&1 &
	nohup ./$(BINARY) -config $(UPPER_240_CFG) >> bot.upper.240.log 2>&1 &
	nohup ./$(BINARY) -config $(UPPER_D_CFG) >> bot.upper.D.log 2>&1 &
	nohup ./$(BINARY) -config $(LOWER_240_CFG) >> bot.lower.240.log 2>&1 &
	nohup ./$(BINARY) -config $(LOWER_D_CFG) >> bot.lower.D.log 2>&1 &
	echo "Все 6 ботов запущены"

run: run-upper-60
run-lower: run-lower-240

stop-upper-60:
	pkill -f "^./$(BINARY) -config $(UPPER_60_CFG)$$" || true
	echo "Upper 1h бот остановлен"

stop-upper-15:
	pkill -f "^./$(BINARY) -config $(UPPER_15_CFG)$$" || true
	echo "Upper 15m бот остановлен"

stop-upper-240:
	pkill -f "^./$(BINARY) -config $(UPPER_240_CFG)$$" || true
	echo "Upper 4h бот остановлен"

stop-upper-d:
	pkill -f "^./$(BINARY) -config $(UPPER_D_CFG)$$" || true
	echo "Upper 1D бот остановлен"

stop-lower-240:
	pkill -f "^./$(BINARY) -config $(LOWER_240_CFG)$$" || true
	echo "Lower 4h бот остановлен"

stop-lower-d:
	pkill -f "^./$(BINARY) -config $(LOWER_D_CFG)$$" || true
	echo "Lower 1D бот остановлен"

stop-upper: stop-upper-60
stop-lower: stop-lower-240

stop:
	$(MAKE) stop-upper-60
	$(MAKE) stop-upper-15
	$(MAKE) stop-upper-240
	$(MAKE) stop-upper-d
	$(MAKE) stop-lower-240
	$(MAKE) stop-lower-d
	echo "Все 6 ботов остановлены"

restart: stop run-all

logs-upper-60:
	tail -f bot.upper.60.log

logs-upper-15:
	tail -f bot.upper.15.log

logs-upper-240:
	tail -f bot.upper.240.log

logs-upper-d:
	tail -f bot.upper.D.log

logs-lower-240:
	tail -f bot.lower.240.log

logs-lower-d:
	tail -f bot.lower.D.log

logs: logs-upper-60
logs-lower: logs-lower-240

status:
	pgrep -f $(BINARY) && echo "✅ Бот работает" || echo "❌ Бот не работает"

clean:
	rm -f $(BINARY) bot.upper.60.log bot.upper.15.log bot.upper.240.log bot.upper.D.log bot.lower.240.log bot.lower.D.log bot.lower.60.log bot.log bot.lower.log
