package main

import (
	"database/sql"
	"flag"
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

var (
	dryRun          bool
	purgeFakeEvents bool
)

func main() {
	flag.BoolVar(&dryRun, "dry-run", false, "Do not commit database transaction")
	flag.BoolVar(&purgeFakeEvents, "purge-fake-events", false, "Delete all previously created fake events")
	flag.Parse()

	conn, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln(err)
	}
	tx, err := conn.Begin()
	if err != nil {
		log.Fatalln(err)
	}

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

	if purgeFakeEvents {
		if err := deleteFakeEvents(tx); err != nil {
			log.Fatalln(err)
		}
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

	if !dryRun {
		err = tx.Commit()
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		fmt.Println("This is a dry run, not committing")
	}
}
