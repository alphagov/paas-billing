package main

import (
	"fmt"
	"os"
	"os/signal"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

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

	sheetsTargetSheet, found := os.LookupEnv("GOOGLE_SHEETS_TARGET_SHEET_ID")
	if !found {
		logger.Error("startup", fmt.Errorf("GOOGLE_SHEETS_TARGET_SHEET_ID environment variable must be set"))
		os.Exit(1)
	}

	googleAPICredentials, found := os.LookupEnv("GOOGLE_API_CREDENTIALS")
	if !found {
		logger.Error("startup", fmt.Errorf("GOOGLE_API_CREDENTIALS environment variable must be set"))
		os.Exit(1)
	}

	sheetsTargetIndex, found := os.LookupEnv("GOOGLE_SHEETS_TARGET_SHEET_INDEX")
	if !found {
		logger.Error("startup", fmt.Errorf("GOOGLE_SHEETS_TARGET_SHEET_INDEX environment variable must be set"))
		os.Exit(1)
	}

	logger.Info("configuration", lager.Data{
		"schedule":     exportFrequency,
		"target_sheet": sheetsTargetSheet,
		"target_index": sheetsTargetIndex,
	})

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

		logSess.Info("create-sheets-client")
		sheets, err := newSheetsService(googleAPICredentials, logSess)
		if err != nil {
			logSess.Error("create-sheets-client", err)
			return
		}

		logSess.Info("clear-sheet", lager.Data{"sheet_id": sheetsTargetSheet})
		err = clearSheet(sheets, sheetsTargetSheet, sheetsTargetIndex)
		if err != nil {
			logSess.Error("clear-sheet", err)
			return
		}

		logSess.Info("write-csv-to-sheet", lager.Data{"sheet_id": sheetsTargetSheet})
		err = writeCSVToSheet(sheets, sheetsTargetSheet, sheetsTargetIndex, csv)
		if err != nil {
			logSess.Error("write-csv-to-sheet", err)
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
