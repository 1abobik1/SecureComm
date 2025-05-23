package utils

import "errors"

var (
	ErrValidatePlatform = errors.New("the error checking platform. available platforms: web, tg-bot")
)

func ValidatePlatform(platform string) error {
	if platform != "web" && platform != "tg-bot" {
		return ErrValidatePlatform
	}
	return nil
}