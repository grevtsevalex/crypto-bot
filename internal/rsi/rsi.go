// Package rsi содержит расчёт RSI (по Уайлдеру, как на Bybit/TradingView) и Stochastic RSI.
package rsi

// RSI по Уайлдеру: первый RSI по SMA за period баров, далее сглаживание
// avgGain_new = (prevAvgGain*13 + currentGain)/14, avgLoss_new = (prevAvgLoss*13 + currentLoss)/14.
func rsiWilder(closes []float64, period int) []float64 {
	n := len(closes)
	if n < period+1 {
		return nil
	}
	out := make([]float64, 0, n-period)

	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			avgGain += diff
		} else {
			avgLoss -= diff
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := period; i < n; i++ {
		var rsi float64
		if avgLoss == 0 {
			rsi = 100
		} else {
			rs := avgGain / avgLoss
			rsi = 100 - (100 / (1 + rs))
		}
		out = append(out, rsi)

		if i == n-1 {
			break
		}
		diff := closes[i+1] - closes[i]
		var g, l float64
		if diff > 0 {
			g = diff
		} else {
			l = -diff
		}
		avgGain = (avgGain*float64(period-1) + g) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + l) / float64(period)
	}

	return out
}

// CalcStochRSI считает Stochastic RSI по стандартной формуле (как на Bybit/TradingView):
// 1) Серия RSI с сглаживанием Уайлдера (период 14).
// 2) Stoch RSI = (текущий RSI − min(RSI за stochPeriod)) / (max − min) × 100.
// closes — цены закрытия, старые → новые. rsiPeriod и stochPeriod обычно 14.
func CalcStochRSI(closes []float64, rsiPeriod, stochPeriod int) float64 {
	rsiSeries := rsiWilder(closes, rsiPeriod)
	if len(rsiSeries) < stochPeriod {
		return 0
	}
	last := len(rsiSeries) - 1
	window := rsiSeries[last+1-stochPeriod : last+1]

	cur := window[stochPeriod-1]
	minR, maxR := window[0], window[0]
	for _, v := range window {
		if v < minR {
			minR = v
		}
		if v > maxR {
			maxR = v
		}
	}
	if maxR == minR {
		return 50
	}
	return (cur - minR) / (maxR - minR) * 100
}
