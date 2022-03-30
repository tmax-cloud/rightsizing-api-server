package main

import (
	"fmt"
	"os"

	"rightsizing-api-server/cmd/api-server/app"
	"rightsizing-api-server/cmd/api-server/app/options"
	_ "rightsizing-api-server/docs"
	log "rightsizing-api-server/internal/logger"
)

func main() {
	option, err := options.NewOptions()
	if err != nil {
		fmt.Print(option.Usage(err))
		os.Exit(1)
	}

	logger, err := log.SetupLogger(*option.LogFile, *option.Mode)
	if err != nil {
		os.Exit(1)
	}

	if err := app.Run(option, logger); err != nil {
		os.Exit(1)
	}
}
