package util

import (
	"time"

	"github.com/rs/zerolog/log"
)

func Retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		log.Error().Msgf("Retrying after %v. An error occurred: %s", delay, err)
		time.Sleep(delay)
	}
	return err
}