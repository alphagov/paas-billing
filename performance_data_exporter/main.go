package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/joho/sqltocsv"
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

	workingDir, _ := os.Getwd()
	scheduler := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger), // or use cron.DefaultLogger
	))

	logger.Info("add-to-schedule", lager.Data{"schedule": exportFrequency})
	_, err := scheduler.AddFunc(exportFrequency, func() {
		logSess := logger.Session("exporter-run")
		exportCSVToStdOut(connectionString, workingDir, logSess)
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

func exportCSVToStdOut(connectionString string, workingDir string, logger lager.Logger) {
	logger.Info("connect-to-db")
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logger.Error("connect-to-db", err)
		return
	}
	defer (func(){
		logger.Info("close-db")
		err := db.Close()
		if err != nil {
			logger.Error("close-db", err)
		}
	})()
	defer cleanup(db, workingDir, logger)

	recreateFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-recreate.sql")
	if err != nil {
		logger.Error("read-table-create-file", err)
		return
	}

	logger.Info("create-database-tables")
	recreateString := string(recreateFileBytes)
	_, err = db.Exec(recreateString)
	if err != nil {
		logger.Error("create-database-tables", err)
		return
	}

	queryFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-generate-csv.sql")
	if err != nil {
		logger.Error("read-query-file", err)
		return
	}

	logger.Error("run-query", err)
	queryString := string(queryFileBytes)
	rows, err := db.Query(queryString)
	if err != nil {
		logger.Error("run-query", err)
		return
	}

	logger.Info("write-to-stdout")
	writer := strings.Builder{}
	err = sqltocsv.Write(&writer, rows)
	if err != nil {
		logger.Error("write-to-stdout", err)
		return
	}
	fmt.Printf("%v", writer.String())
}

func cleanup(db *sql.DB, workingDir string, logger lager.Logger) {
	cleanupFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-recreate.sql")
	if err != nil {
		logger.Error("read-cleanup-file", err)
		return
	}
	cleanupString := string(cleanupFileBytes)

	logger.Info("perform-database-cleanup")
	_, err = db.Exec(cleanupString)
	if err != nil {
		logger.Error("perform-database-cleanup", err)
		return
	}
}
