# Crypto RSI/Stoch RSI Bot

Telegram-бот по криптопарам Bybit: считает **RSI** и **Stochastic RSI** и отправляет уведомления либо по **верхней зоне**, либо по **нижней зоне**. Режим выбирается в конфиге, поэтому из одного репозитория можно запускать два отдельных бота с разными токенами и разными файлами подписчиков. Пользователь может менять только таймфрейм свечей, а формула индикаторов зафиксирована на канонических значениях Bybit/TradingView.

## Как это устроено

1. **Таймфрейм** — используются свечи Bybit выбранного интервала.
2. **Расчёт индикаторов** — RSI по Уайлдеру, затем `raw Stoch RSI`, затем сглаживание `%K/%D`.
3. **Сигнал upper** — если одновременно выполнены условия `RSI ≥ 70` и `Stoch RSI %K ≥ 99.99`.
4. **Сигнал lower** — если одновременно выполнены условия `RSI ≤ 30` и `Stoch RSI %K = 0`.
5. **Подписчики** каждого бота хранятся в своём JSON-файле.

## Требования

- Go 1.24+
- Токен бота от [@BotFather](https://t.me/BotFather)

## Установка и запуск

```bash
cd crypto-bot
go build -o crypto-bot .

# Указать telegram_token в config.json (или в отдельном config для второго бота)
./crypto-bot
```

При первом запуске без `config.json` создаётся файл с полями по умолчанию. Нужно заполнить `telegram_token`.

## Запуск двух ботов

Пример для двух отдельных Telegram-ботов:

```bash
cp config.example.json config.upper.json
cp config.lower.example.json config.lower.json

# Вставить разные telegram_token
./crypto-bot -config config.upper.json
./crypto-bot -config config.lower.json
```

Рекомендуется:

- для `upper` использовать `subscribers.json`
- для `lower` использовать `subscribers.lower.json`

## Конфигурация

Поля в `config.json` (дефолты как в `config.example.json`):

| Параметр                 | Описание                          | По умолчанию |
|--------------------------|-----------------------------------|--------------|
| `telegram_token`         | Токен бота                        | —            |
| `subscribers_file`       | Файл подписчиков                  | `subscribers.json` |
| `signal_mode`            | Режим сигнала: `upper` или `lower` | `upper` |
| `timeframe`              | Таймфрейм свечей Bybit (`5`, `15`, `60`, `240`, `D`) | `60` |
| `max_signals_per_cycle`  | Макс. уведомлений за проход       | 10           |
| `candle_limit`           | Число часовых свечей              | 100          |

В боте через **/settings** меняется только таймфрейм. Все индикаторные параметры зафиксированы.

## Команды бота

| Команда     | Действие         |
|-------------|------------------|
| `/start`    | Главное меню     |
| `/settings` | Настройки        |
| `/status`   | Статус подписки  |
| `/stop`     | Отписаться       |
| `/help`     | Справка          |

## Параметры расчёта (зашиты в коде)

- **Таймфрейм:** 1h (60 мин)
- **Доступные таймфреймы:** `5`, `15`, `60`, `240`, `D`
- **Свечей:** 100
- **RSI период:** 14 (сглаживание Уайлдера / RMA)
- **Stoch период:** 14 (окно min/max RSI)
- **Stoch smoothing:** `%K = SMA(3)`, `%D = SMA(3)`
- **RSI upper threshold:** 70
- **RSI lower threshold:** 30
- **Stoch RSI %K upper threshold:** 99.99
- **Stoch RSI %K lower threshold:** 0

## Как считаются показатели

1. `RSI` считается по классической формуле Уайлдера.
2. Первый `avgGain` и `avgLoss` берутся как `SMA` за `14` баров.
3. Далее используется сглаживание Уайлдера:

```text
avgGain = (prevAvgGain * (period - 1) + currentGain) / period
avgLoss = (prevAvgLoss * (period - 1) + currentLoss) / period
RSI = 100 - 100 / (1 + avgGain / avgLoss)
```

4. `raw Stoch RSI` считается не по цене, а по уже полученному ряду `RSI`:

```text
rawStochRSI = (RSI_now - min(RSI, 14)) / (max(RSI, 14) - min(RSI, 14)) * 100
```

5. Затем применяется сглаживание:

```text
%K = SMA(rawStochRSI, 3)
%D = SMA(%K, 3)
```

6. Для верхнего бота сигнал:

```text
RSI >= 70
AND
Stoch RSI %K >= 99.99
```

7. Для нижнего бота сигнал:

```text
RSI <= 30
AND
Stoch RSI %K <= 0
```

Канонические настройки для соответствия графику Bybit/TradingView: `RSI 14`, `Stoch 14`, `smooth_k = 3`, `smooth_d = 3`.

## Структура проекта

```
crypto-bot/
├── main.go                 # Точка входа, цикл анализа по выбранному таймфрейму
├── config.json
├── config.example.json
├── subscribers.json
└── internal/
    ├── config/             # Telegram token, режим сигнала и настройки запуска
    ├── exchange/           # Список пар и свечи Bybit
    ├── handlers/           # Подписка, отписка, статус, справка
    ├── notify/             # Рассылка при верхней или нижней зоне RSI/Stoch RSI
    └── rsi/                # RSI по Уайлдеру + Stoch RSI (%K/%D)
```

Не передавайте `telegram_token` в публичные репозитории; при необходимости добавьте `config.json` в `.gitignore`.
