package main

import (
	"context"
	"log/slog"
	"os"
	"terminalui/tui"
)

func main() {
	ctx := context.Background()
	logFileName := "./app.log"
	logFile, fileErr := os.Create(logFileName)
	if fileErr != nil {
		panic(fileErr)
	}

	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{}))
	tui.DisplayUI(ctx, logger)
}
