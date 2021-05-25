package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

type CSVOutputter interface {
	WriteCSV(csv string) error
}

func main() {
	logger := lager.NewLogger("performance-data-exporter")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	logger.Info("startup")
	connectionString, found := os.LookupEnv("DATABASE_URL")
	if !found {
		logger.Error("startup", fmt.Errorf("DATABASE_URL environment variable must be set"))
		os.Exit(1)
	}

	exportFrequency, found := os.LookupEnv("EXPORT_FREQUENCY")
	if !found {
		logger.Error("startup", fmt.Errorf("EXPORT_FREQUENCY environment variable must be set"))
		os.Exit(1)
	}

	outputTarget, found := os.LookupEnv("OUTPUT_TARGET")
	if !found {
		logger.Error("startup", fmt.Errorf("OUTPUT_TARGET environment variable must be set"))
		os.Exit(1)
	}

	lagerConfigMessageData := lager.Data{"schedule": exportFrequency}
	var outputter CSVOutputter
	switch strings.ToUpper(outputTarget) {
	case "GOOGLE_SHEETS":
		googleAPICredentials, found := os.LookupEnv("GOOGLE_API_CREDENTIALS")
		if !found {
			logger.Error("startup", fmt.Errorf("GOOGLE_API_CREDENTIALS environment variable must be set"))
			os.Exit(1)
		}

		sheetsTargetSheet, found := os.LookupEnv("GOOGLE_SHEETS_TARGET_SHEET_ID")
		if !found {
			logger.Error("startup", fmt.Errorf("GOOGLE_SHEETS_TARGET_SHEET_ID environment variable must be set"))
			os.Exit(1)
		}

		sheetsTargetIndexstr, found := os.LookupEnv("GOOGLE_SHEETS_TARGET_SHEET_INDEX")
		if !found {
			logger.Error("startup", fmt.Errorf("GOOGLE_SHEETS_TARGET_SHEET_INDEX environment variable must be set"))
			os.Exit(1)
		}
		sheetsTargetIndex, err := strconv.ParseInt(sheetsTargetIndexstr, 10, 64)
		if err != nil {
			logger.Error("startup", fmt.Errorf("GOOGLE_SHEETS_TARGET_SHEET_INDEX environment variable must be an integer"))
			os.Exit(1)
		}
		outputter = NewGoogleSheetsOutputter(googleAPICredentials, sheetsTargetSheet, sheetsTargetIndex, logger)

		lagerConfigMessageData["target_sheet"] = sheetsTargetSheet
		lagerConfigMessageData["target_index"] = sheetsTargetIndex

	case "STDOUT":
		outputter = NewStdOutOutputter(logger)
	default:
		logger.Error("startup", fmt.Errorf("OUTPUT_TARGET must be one of GOOGLE_SHEETS, STDOUT"))
		os.Exit(1)
	}
	lagerConfigMessageData["output_target"] = outputTarget
	logger.Info("configuration", lagerConfigMessageData)

	workingDir, _ := os.Getwd()
	scheduler := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger), // or use cron.DefaultLogger
	))

	logger.Info("add-to-schedule", lager.Data{"schedule": exportFrequency})
	_, err := scheduler.AddFunc(exportFrequency, func() {
		logSess := logger.Session("exporter-run")

		logSess.Info("generate-csv")
		csv, err := generateCSV(connectionString, workingDir, logSess)
		if err != nil {
			logSess.Error("generate-csv", err)
			return
		}

		logSess.Info("write-csv")
		err = outputter.WriteCSV(csv)
		if err != nil {
			logSess.Error("write-csv", err)
			return
		}
	})
	if err != nil {
		logger.Error("add-to-schedule", err)
		os.Exit(1)
	}

	logger.Info("begin-schedule")
	scheduler.Start()
	waitForExit()
}

func waitForExit() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	return <-c
}
