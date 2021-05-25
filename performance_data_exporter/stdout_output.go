package main

import (
	"code.cloudfoundry.org/lager"
	"fmt"
)

type StdOutOutputter struct {
	Logger lager.Logger
}

func NewStdOutOutputter(logger lager.Logger) *StdOutOutputter {
	return &StdOutOutputter{Logger: logger.Session("std-out-outputter")}
}

func (s StdOutOutputter) WriteCSV(csv string) error {
	s.Logger.Session("write-csv").Info("write-csv")
	fmt.Println(csv)
	return nil
}

