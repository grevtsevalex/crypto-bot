// Package exchange содержит обращение к API бирж: список торговых пар (Bybit linear) и свечи (цены закрытия) для расчёта RSI.
package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	bybitInstrumentsPath = "/v5/market/instruments-info?category=linear"
	bybitKlinePathFmt    = "/v5/market/kline?category=linear&symbol=%s&interval=%s&limit=%d"
)

var bybitMainnetHosts = []string{
	"https://api.bybit.com",
	"https://api.bytick.com",
	"https://api.bybit.kz",
	"https://api.bybit-tr.com",
	"https://api.bybit.ae",
}

// DerivativePairs возвращает список символов линейных деривативов Bybit (category=linear) в статусе Trading.
func DerivativePairs() ([]string, error) {
	body, err := bybitGETAny(bybitInstrumentsPath, 15*time.Second)
	if err != nil {
		return nil, err
	}

	var data struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Symbol string `json:"symbol"`
				Status string `json:"status"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа Bybit (pairs): %w", err)
	}
	if data.RetCode != 0 {
		return nil, fmt.Errorf("bybit pairs retCode=%d retMsg=%s", data.RetCode, data.RetMsg)
	}

	var result []string
	for _, s := range data.Result.List {
		if s.Status == "Trading" {
			result = append(result, s.Symbol)
		}
	}
	return result, nil
}

// Candles запрашивает свечи с Bybit (linear) и возвращает цены закрытия в хронологическом порядке (старые → новые).
// symbol — тикер, например BTCUSDT; timeframe — интервал: "5", "15", "60", "240" и т.д.; limit — число свечей.
func Candles(symbol, timeframe string, limit int) ([]float64, error) {
	body, err := bybitGETAny(fmt.Sprintf(bybitKlinePathFmt, symbol, timeframe, limit), 10*time.Second)
	if err != nil {
		return nil, err
	}

	var data struct {
		Result struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа Bybit (candles): %w", err)
	}

	var closes []float64
	for i := len(data.Result.List) - 1; i >= 0; i-- {
		closePrice, err := strconv.ParseFloat(data.Result.List[i][4], 64)
		if err != nil {
			continue
		}
		closes = append(closes, closePrice)
	}

	return closes, nil
}

// bybitGETAny пробует выполнить запрос к нескольким официальным mainnet-доменам Bybit.
// Это нужно, потому что некоторые регионы/сети могут получать 403 на api.bybit.com.
func bybitGETAny(pathAndQuery string, timeout time.Duration) ([]byte, error) {
	var lastErr error
	for _, host := range bybitMainnetHosts {
		url := host + pathAndQuery
		body, err := bybitGET(url, timeout)
		if err == nil {
			return body, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("не удалось получить ответ Bybit ни с одного домена: %w", lastErr)
}

// bybitGET выполняет GET-запрос к Bybit c базовыми заголовками и проверкой,
// что пришёл JSON-ответ с HTTP 200. Если приходит HTML (например, блокировка/ошибка),
// возвращает понятную ошибку с фрагментом тела.
func bybitGET(url string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Делаем запрос максимально похожим на успешный curl из Postman.
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "curl/8.7.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("bybit http %d: %s", resp.StatusCode, snippet)
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, errors.New("пустой ответ bybit")
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		if len(trimmed) > 200 {
			trimmed = trimmed[:200]
		}
		return nil, fmt.Errorf("ожидался json от bybit, получено: %s", trimmed)
	}
	return body, nil
}
