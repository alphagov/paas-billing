package compose

import (
	"fmt"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

//go:generate counterfeiter -o fakes/fake_client.go . Client
type Client interface {
	GetAuditEvents(params composeapi.AuditEventsParams) (*[]composeapi.AuditEvent, []error)
}

func NewClient(apiToken string) (Client, error) {
	return composeapi.NewClient(apiToken)
}

func SquashErrors(errs []error) error {
	var s []string

	for _, err := range errs {
		s = append(s, err.Error())
	}

	return fmt.Errorf("%s", strings.Join(s, "; "))
}
