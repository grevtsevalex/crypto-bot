package rsi

// Calc считает RSI по ценам закрытия и периоду.
// Возвращает 0, если данных недостаточно (меньше period+1 свечей).
func Calc(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}

	var gain, loss float64
	for i := 1; i <= period; i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}

	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}
