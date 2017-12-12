package main

import (
	"log"
	"os"

	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	"github.com/alphagov/paas-usage-events-collector/db"
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
	sqlClient, err := db.NewPostgresClient(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln(err)
	}

	cfClient, err := createCFClient()
	if err != nil {
		log.Fatalln(err)
	}

	err = sqlClient.RepairEvents(cfClient)
	if err != nil {
		log.Fatalln(err)
	}
}
