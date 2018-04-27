package composefetcher

import (
	"errors"
	"fmt"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

type ComposeClient interface {
	GetAuditEvents(params composeapi.AuditEventsParams) (*[]composeapi.AuditEvent, []error)
}

func newClient(apiToken string) (ComposeClient, error) {
	if apiToken == "" {
		return nil, errors.New("Compose API token is required")
	}
	return composeapi.NewClient(apiToken)
}

func squashErrors(errs []error) error {
	var s []string

	for _, err := range errs {
		s = append(s, err.Error())
	}

	return fmt.Errorf("%s", strings.Join(s, "; "))
}
