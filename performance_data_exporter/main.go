package main

import (
	"database/sql"
	"fmt"
	"github.com/joho/sqltocsv"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	connectionString, found := os.LookupEnv("DATABASE_URL")
	if !found {
		log.Fatalln("DATABASE_URL environment variable must be set")
	}
	workingDir, _ := os.Getwd()

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

	queryFileBytes, err := ioutil.ReadFile(workingDir+"/sql/usage-and-adoption-generate-csv.sql")
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
