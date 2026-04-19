package env

import (
	"fmt"
	"os"
)

type Mode string

var (
	ModeProd    Mode = "production"
	ModeFeedDev Mode = "feedDev"
	ModeDev     Mode = "dev"
)

func GetMode() (Mode, error) {
	env := os.Getenv("ENV")
	switch env {
	case "production":
		return ModeProd, nil
	case "feedDev":
		return ModeFeedDev, nil
	case "dev", "local":
		return ModeDev, nil
	default:
		return "", fmt.Errorf("unrecognized mode: %s", env)
	}
}
