package main

import (
	"code.cloudfoundry.org/lager"
	"database/sql"
	"github.com/joho/sqltocsv"
	"io/ioutil"
	"strings"
)

func generateCSV(connectionString string, workingDir string, logger lager.Logger) (string, error) {
	logger.Info("connect-to-db")
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logger.Error("connect-to-db", err)
		return "", err
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
		return "", err
	}

	logger.Info("create-database-tables")
	recreateString := string(recreateFileBytes)
	_, err = db.Exec(recreateString)
	if err != nil {
		logger.Error("create-database-tables", err)
		return "", err
	}

	queryFileBytes, err := ioutil.ReadFile(workingDir + "/sql/usage-and-adoption-generate-csv.sql")
	if err != nil {
		logger.Error("read-query-file", err)
		return "", err
	}

	logger.Error("run-query", err)
	queryString := string(queryFileBytes)
	rows, err := db.Query(queryString)
	if err != nil {
		logger.Error("run-query", err)
		return "", err
	}

	logger.Info("write-to-stdout")
	writer := strings.Builder{}
	err = sqltocsv.Write(&writer, rows)
	if err != nil {
		logger.Error("write-to-stdout", err)
		return "", err
	}
	return writer.String(), nil
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
