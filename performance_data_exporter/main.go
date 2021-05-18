package main

import (
	"database/sql"
	"fmt"
	"github.com/joho/sqltocsv"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
)

func main() {
	connectionString, found := os.LookupEnv("DATABASE_URL")
	if !found {
		log.Fatalln("DATABASE_URL environment variable must be set")
	}

	exportFrequency, found := os.LookupEnv("EXPORT_FREQUENCY")
	if !found {
		log.Fatalln("EXPORT_FREQUENCY environment variable must be set")
	}

	workingDir, _ := os.Getwd()

	scheduler := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),  // or use cron.DefaultLogger
	))
	_, err := scheduler.AddFunc(exportFrequency, func() {
		exportCSVToStdOut(connectionString, workingDir)
	})
	if err != nil {
		log.Fatalln(err)
	}

	scheduler.Start()
	waitForExit()
}

func waitForExit() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	return <-c
}

func exportCSVToStdOut(connectionString string, workingDir string) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	defer cleanup(db, workingDir)

	recreateFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-recreate.sql")
	if err != nil {
		log.Fatalln(err)
	}
	recreateString := string(recreateFileBytes)
	_, err = db.Exec(recreateString)
	if err != nil {
		log.Fatalln(err)
	}

	queryFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-generate-csv.sql")
	if err != nil {
		log.Fatalln(err)
	}

	queryString := string(queryFileBytes)
	rows, err := db.Query(queryString)
	if err != nil {
		log.Fatalln(err)
	}

	writer := strings.Builder{}
	err = sqltocsv.Write(&writer, rows)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%v", writer.String())
}

func cleanup(db *sql.DB, workingDir string) {
	cleanupFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-recreate.sql")
	if err != nil {
		log.Fatalln(err)
	}
	cleanupString := string(cleanupFileBytes)
	_, err = db.Exec(cleanupString)
	if err != nil {
		log.Fatalln(err)
	}
}
