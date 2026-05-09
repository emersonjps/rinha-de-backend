package engine

import (
	"errors"

	"github.com/buger/jsonparser"
)

// Hardcoded normalization constants for speed.
const (
	MaxAmount            float32 = 10000.0
	MaxInstallments      float32 = 12.0
	AmountVsAvgRatio     float32 = 10.0
	MaxMinutes           float32 = 1440.0
	MaxKm                float32 = 1000.0
	MaxTxCount24h        float32 = 20.0
	MaxMerchantAvgAmount float32 = 10000.0
)

var (
	ErrInvalidPayload   = errors.New("invalid payload")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)

func clamp(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Vectorize converts a JSON payload into a 14-dimension feature vector.
func Vectorize(body []byte, mccRisks map[string]float32) ([14]float32, error) {
	var v [14]float32

	transaction, err := getObject(body, "transaction")
	if err != nil {
		return v, err
	}
	customer, err := getObject(body, "customer")
	if err != nil {
		return v, err
	}
	merchant, err := getObject(body, "merchant")
	if err != nil {
		return v, err
	}
	terminal, err := getObject(body, "terminal")
	if err != nil {
		return v, err
	}

	amount, err := jsonparser.GetFloat(transaction, "amount")
	if err != nil {
		return v, ErrInvalidPayload
	}
	v[0] = clamp(float32(amount) / MaxAmount)

	installments, err := jsonparser.GetInt(transaction, "installments")
	if err != nil {
		return v, ErrInvalidPayload
	}
	v[1] = clamp(float32(installments) / MaxInstallments)

	avgAmount, err := jsonparser.GetFloat(customer, "avg_amount")
	if err != nil {
		return v, ErrInvalidPayload
	}
	if avgAmount > 0 {
		v[2] = clamp(float32(amount/avgAmount) / AmountVsAvgRatio)
	} else {
		v[2] = 0
	}

	requestedAt, err := jsonparser.GetUnsafeString(transaction, "requested_at")
	if err != nil {
		return v, ErrInvalidPayload
	}
	year, month, day, hour, minute, second, ok := parseISO8601(requestedAt)
	if !ok {
		return v, ErrInvalidTimestamp
	}
	v[3] = float32(hour) / 23.0
	weekday := weekdayFromDate(year, month, day)
	v[4] = float32(weekday) / 6.0

	reqSeconds := unixSeconds(year, month, day, hour, minute, second)

	lastTx, lastTxType, _, err := jsonparser.Get(body, "last_transaction")
	if err != nil {
		return v, ErrInvalidPayload
	}
	if lastTxType == jsonparser.Null {
		v[5] = -1
		v[6] = -1
	} else if lastTxType == jsonparser.Object {
		lastTimestamp, err := jsonparser.GetUnsafeString(lastTx, "timestamp")
		if err != nil {
			return v, ErrInvalidPayload
		}
		lyear, lmonth, lday, lhour, lminute, lsecond, ok := parseISO8601(lastTimestamp)
		if !ok {
			return v, ErrInvalidTimestamp
		}
		lastSeconds := unixSeconds(lyear, lmonth, lday, lhour, lminute, lsecond)
		deltaSeconds := reqSeconds - lastSeconds
		if deltaSeconds < 0 {
			deltaSeconds = 0
		}
		minutes := float32(deltaSeconds) / 60.0
		v[5] = clamp(minutes / MaxMinutes)

		kmFromLast, err := jsonparser.GetFloat(lastTx, "km_from_current")
		if err != nil {
			return v, ErrInvalidPayload
		}
		v[6] = clamp(float32(kmFromLast) / MaxKm)
	} else {
		return v, ErrInvalidPayload
	}

	kmFromHome, err := jsonparser.GetFloat(terminal, "km_from_home")
	if err != nil {
		return v, ErrInvalidPayload
	}
	v[7] = clamp(float32(kmFromHome) / MaxKm)

	txCount24h, err := jsonparser.GetInt(customer, "tx_count_24h")
	if err != nil {
		return v, ErrInvalidPayload
	}
	v[8] = clamp(float32(txCount24h) / MaxTxCount24h)

	isOnline, err := jsonparser.GetBoolean(terminal, "is_online")
	if err != nil {
		return v, ErrInvalidPayload
	}
	if isOnline {
		v[9] = 1
	} else {
		v[9] = 0
	}

	cardPresent, err := jsonparser.GetBoolean(terminal, "card_present")
	if err != nil {
		return v, ErrInvalidPayload
	}
	if cardPresent {
		v[10] = 1
	} else {
		v[10] = 0
	}

	merchantID, err := jsonparser.GetUnsafeString(merchant, "id")
	if err != nil {
		return v, ErrInvalidPayload
	}

	unknownMerchant := float32(1)
	knownMerchants, kmType, _, err := jsonparser.Get(customer, "known_merchants")
	if err == nil && kmType == jsonparser.Array {
		_, _ = jsonparser.ArrayEach(knownMerchants, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if unknownMerchant == 0 || dataType != jsonparser.String {
				return
			}
			if equalBytesString(value, merchantID) {
				unknownMerchant = 0
			}
		})
	}
	v[11] = unknownMerchant

	mcc, err := jsonparser.GetUnsafeString(merchant, "mcc")
	if err != nil {
		return v, ErrInvalidPayload
	}
	if risk, ok := mccRisks[mcc]; ok {
		v[12] = risk
	} else {
		v[12] = 0.5
	}

	merchantAvg, err := jsonparser.GetFloat(merchant, "avg_amount")
	if err != nil {
		return v, ErrInvalidPayload
	}
	v[13] = clamp(float32(merchantAvg) / MaxMerchantAvgAmount)

	return v, nil
}

func getObject(body []byte, key string) ([]byte, error) {
	value, dataType, _, err := jsonparser.Get(body, key)
	if err != nil || dataType != jsonparser.Object {
		return nil, ErrInvalidPayload
	}
	return value, nil
}

func parseISO8601(s string) (year, month, day, hour, minute, second int, ok bool) {
	if len(s) < 20 {
		return 0, 0, 0, 0, 0, 0, false
	}
	if s[4] != '-' || s[7] != '-' || s[10] != 'T' || s[13] != ':' || s[16] != ':' {
		return 0, 0, 0, 0, 0, 0, false
	}
	var okPart bool

	year, okPart = parse4(s, 0)
	if !okPart {
		return 0, 0, 0, 0, 0, 0, false
	}
	month, okPart = parse2(s, 5)
	if !okPart {
		return 0, 0, 0, 0, 0, 0, false
	}
	day, okPart = parse2(s, 8)
	if !okPart {
		return 0, 0, 0, 0, 0, 0, false
	}
	hour, okPart = parse2(s, 11)
	if !okPart {
		return 0, 0, 0, 0, 0, 0, false
	}
	minute, okPart = parse2(s, 14)
	if !okPart {
		return 0, 0, 0, 0, 0, 0, false
	}
	second, okPart = parse2(s, 17)
	if !okPart {
		return 0, 0, 0, 0, 0, 0, false
	}

	return year, month, day, hour, minute, second, true
}

func parse2(s string, i int) (int, bool) {
	if i+1 >= len(s) {
		return 0, false
	}
	a := s[i]
	b := s[i+1]
	if a < '0' || a > '9' || b < '0' || b > '9' {
		return 0, false
	}
	return int(a-'0')*10 + int(b-'0'), true
}

func parse4(s string, i int) (int, bool) {
	if i+3 >= len(s) {
		return 0, false
	}
	a := s[i]
	b := s[i+1]
	c := s[i+2]
	d := s[i+3]
	if a < '0' || a > '9' || b < '0' || b > '9' || c < '0' || c > '9' || d < '0' || d > '9' {
		return 0, false
	}
	return int(a-'0')*1000 + int(b-'0')*100 + int(c-'0')*10 + int(d-'0'), true
}

func unixSeconds(year, month, day, hour, minute, second int) int64 {
	days := daysSinceCivil(year, month, day)
	return ((days*24+int64(hour))*60+int64(minute))*60 + int64(second)
}

func weekdayFromDate(year, month, day int) int {
	days := daysSinceCivil(year, month, day)
	weekday := int((days + 3) % 7)
	if weekday < 0 {
		weekday += 7
	}
	return weekday
}

func daysSinceCivil(year, month, day int) int64 {
	if month <= 2 {
		year--
		month += 12
	}
	era := floorDiv(year, 400)
	yoe := year - era*400
	doy := (153*(month-3)+2)/5 + day - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return int64(era*146097+doe-719468)
}

func floorDiv(a, b int) int {
	if a >= 0 {
		return a / b
	}
	return -(((-a) + b - 1) / b)
}

func equalBytesString(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := 0; i < len(b); i++ {
		if b[i] != s[i] {
			return false
		}
	}
	return true
}
