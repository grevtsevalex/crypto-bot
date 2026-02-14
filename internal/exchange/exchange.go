// Package exchange содержит обращение к API бирж: список торговых пар и свечи (цены закрытия).
package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	binanceExchangeInfoURL = "https://api.binance.com/api/v3/exchangeInfo"
	bybitKlineURL          = "https://api.bybit.com/v5/market/kline"
)

// DerivativePairs получить деривативы
func DerivativePairs() ([]string, error) {
    url := "https://api.bybit.com/v5/market/instruments-info?category=linear"

    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var data struct {
        Result struct {
            List []struct {
                Symbol string `json:"symbol"`
                Status string `json:"status"`
            } `json:"list"`
        } `json:"result"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, err
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
// timeframe — интервал в минутах: "1", "5", "15", "60" и т.д.
func Candles(symbol, timeframe string, limit int) ([]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s?category=linear&symbol=%s&interval=%s&limit=%d",
		bybitKlineURL, symbol, timeframe, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Result struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
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
