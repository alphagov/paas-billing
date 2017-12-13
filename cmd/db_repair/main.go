package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
)

/*
DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB' \
CF_API_ADDRESS="$(cf target | awk '/^api endpoint/ {print $3}')" \
cf target | awk '/^api endpoint/ {print $3}' \
go run cmd/db_repair/main.go
*/

func createCFClient() (cloudfoundry.Client, error) {
	config := cloudfoundry.CreateConfigFromEnv()
	return cloudfoundry.NewClient(config)
}

func main() {
	conn, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln(err)
	}
	tx, err := conn.Begin()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	cfClient, err := createCFClient()
	if err != nil {
		log.Fatalln(err)
	}

	spaces, err := cfClient.GetSpaces()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("spaces:", len(spaces))

	epoch, err := getCollectionEpoch(tx)
	if err != nil {
		log.Fatalln(err)
	}

	err = createEventsForAppsWithNoRecordedEvents(tx, epoch, spaces, cfClient)
	if err != nil {
		log.Fatalln(err)
	}
	err = createEventsForServicesWithNoRecordedEvents(tx, epoch, spaces, cfClient)
	if err != nil {
		log.Fatalln(err)
	}
	err = createEventsForAppsWhereFirstRecordedEventIsStopped(tx, epoch)
	if err != nil {
		log.Fatalln(err)
	}
	err = createEventsForServicesWhereFirstRecordedEventIsDeleted(tx, epoch)
	if err != nil {
		log.Fatalln(err)
	}
}
