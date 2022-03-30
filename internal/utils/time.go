package utils

import (
	"time"

	"github.com/araddon/dateparse"
)

func TimeParser(datestr string) (time.Time, error) {
	t, err := dateparse.ParseAny(datestr)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func ParseQueryTime(startstr, endstr string) (time.Time, time.Time, error) {
	// start: default a week ago
	var start = time.Now().AddDate(0, 0, -7)
	var end = time.Now()
	var err error

	if startstr != "" {
		if start, err = TimeParser(startstr); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if endstr != "" {
		if end, err = TimeParser(endstr); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return start, end, nil
}
