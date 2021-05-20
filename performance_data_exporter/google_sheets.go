package main

import (
	"context"
	"encoding/json"

	"code.cloudfoundry.org/lager"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type credentials struct {
	ClientEmail string `json:"client_email"`
}

func newSheetsService(googleAPICredentials string, logger lager.Logger) (*sheets.Service, error) {
	credentialBytes := []byte(googleAPICredentials)
	creds := credentials{}
	if err := json.Unmarshal(credentialBytes, &creds); err != nil {
		logger.Error("get-service-account-email", err)
	}
	logger.Info("service-account-email", lager.Data{"email": creds.ClientEmail})
	return sheets.NewService(context.Background(), option.WithCredentialsJSON(credentialBytes))
}

func clearSheet(service *sheets.Service, sheetId string, sheetIndex int64) error {
	// This request looks a little unusual, but matches the documentation.
	//   > Specifying userEnteredValue in fields without providing a corresponding value is
	//   > interpreted as an instruction to clear values in the range
	// https://developers.google.com/sheets/api/samples/sheet#clear_a_sheet_of_all_values_while_preserving_formats
	call := service.Spreadsheets.BatchUpdate(sheetId, &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: false,
		Requests: []*sheets.Request{
			{
				UpdateCells: &sheets.UpdateCellsRequest{
					Fields: "userEnteredValue",
					Range: &sheets.GridRange{
						SheetId: sheetIndex,
					},
				},
			},
		},
	})

	_, err := call.Do()
	return err
}

func writeCSVToSheet(service *sheets.Service, sheetId string, sheetIndex int64, csvData string) error {
	// Import a CSV by performing a batch update with a parse data request.
	// The parse data request will take the string containing the CSV data,
	// parse it as a CSV, and inject it in to the cells staring at given coordinate.
	call := service.Spreadsheets.BatchUpdate(sheetId, &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: false,
		Requests: []*sheets.Request{
			{
				PasteData: &sheets.PasteDataRequest{
					Coordinate: &sheets.GridCoordinate{
						ColumnIndex: 0,
						RowIndex:    0,
						SheetId:     sheetIndex,
					},
					Data:      csvData,
					Delimiter: ",",
					Html:      false,
					Type:      "PASTE_VALUES",
				},
			},
		},
	})

	_, err := call.Do()
	return err
}
