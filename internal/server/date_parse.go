package server

import (
	"errors"
	"strings"
	"time"
)

func parseDateOfBirth(s string) (time.Time, error) {
	v := strings.TrimSpace(s)
	if v == "" {
		return time.Time{}, errors.New("empty date")
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t, nil
	}
	return time.Parse("01/02/2006", v)
}
