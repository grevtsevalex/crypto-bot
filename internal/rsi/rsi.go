// Package rsi содержит расчёт RSI (по Уайлдеру, как на Bybit/TradingView) и Stochastic RSI.
package rsi

// StochRSIValues хранит последние значения осцилляторов в формате,
// близком к отображению на графиках Bybit/TradingView.
type StochRSIValues struct {
	RSI  float64
	RawK float64
	K    float64
	D    float64
}

// RSI по Уайлдеру: первый RSI по SMA за period баров, далее сглаживание
// avgGain_new = (prevAvgGain*13 + currentGain)/14, avgLoss_new = (prevAvgLoss*13 + currentLoss)/14.
func rsiWilder(closes []float64, period int) []float64 {
	n := len(closes)
	if period <= 0 || n < period+1 {
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

// CalcRSI возвращает последнее значение RSI по Уайлдеру.
func CalcRSI(closes []float64, period int) float64 {
	rsiSeries := rsiWilder(closes, period)
	if len(rsiSeries) == 0 {
		return 0
	}
	return rsiSeries[len(rsiSeries)-1]
}

func sma(values []float64, period int) []float64 {
	if period <= 0 || len(values) < period {
		return nil
	}

	out := make([]float64, 0, len(values)-period+1)
	var sum float64
	for i, v := range values {
		sum += v
		if i >= period {
			sum -= values[i-period]
		}
		if i >= period-1 {
			out = append(out, sum/float64(period))
		}
	}
	return out
}

func stoch(values []float64, period int) []float64 {
	if period <= 0 || len(values) < period {
		return nil
	}

	out := make([]float64, 0, len(values)-period+1)
	for i := period - 1; i < len(values); i++ {
		window := values[i+1-period : i+1]
		cur := window[len(window)-1]
		minV, maxV := window[0], window[0]
		for _, v := range window {
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
		if maxV == minV {
			out = append(out, 0)
			continue
		}
		out = append(out, (cur-minV)/(maxV-minV)*100)
	}
	return out
}

// CalcStochRSI считает Stochastic RSI в формате Bybit/TradingView:
// 1) RSI по Уайлдеру.
// 2) Raw Stoch RSI = Stoch(RSI, RSI, RSI, stochPeriod).
// 3) %K = SMA(raw, smoothK), %D = SMA(%K, smoothD).
// closes — цены закрытия, старые → новые. Классические значения: 14/14/3/3.
func CalcStochRSI(closes []float64, rsiPeriod, stochPeriod, smoothK, smoothD int) StochRSIValues {
	rsiSeries := rsiWilder(closes, rsiPeriod)
	if len(rsiSeries) == 0 {
		return StochRSIValues{}
	}

	rawSeries := stoch(rsiSeries, stochPeriod)
	kSeries := sma(rawSeries, smoothK)
	dSeries := sma(kSeries, smoothD)

	values := StochRSIValues{
		RSI: rsiSeries[len(rsiSeries)-1],
	}
	if len(rawSeries) > 0 {
		values.RawK = rawSeries[len(rawSeries)-1]
	}
	if len(kSeries) > 0 {
		values.K = kSeries[len(kSeries)-1]
	} else {
		values.K = values.RawK
	}
	if len(dSeries) > 0 {
		values.D = dSeries[len(dSeries)-1]
	} else {
		values.D = values.K
	}

	return values
}
