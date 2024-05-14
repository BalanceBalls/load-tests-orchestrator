package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"terminalui/tui"
)

const (
	DefaultUpdateIntervalSec = 3
	// 3 days
	DefaultKeepContainerAliveForInSec = 259200
)

var (
	updateInterval int
	keepAlive      int
)

func main() {
	customUpdateInterval := flag.Int("refresh", 3, "refresh rate for logs streaming")
	customKeepAlive := flag.Int("keep-alive", 259200, "keep pods alive for N seconds")
	flag.Parse()

	if customUpdateInterval != nil {
		updateInterval = *customUpdateInterval
	} else {
		updateInterval = DefaultUpdateIntervalSec
	}

	if customKeepAlive != nil {
		keepAlive = *customKeepAlive
	} else {
		keepAlive = DefaultKeepContainerAliveForInSec
	}

	ctx := context.Background()
	logFileName := "./app.log"
	logFile, fileErr := os.Create(logFileName)
	if fileErr != nil {
		panic(fileErr)
	}

	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{}))
	tui.DisplayUI(ctx, logger, updateInterval, keepAlive)
}
