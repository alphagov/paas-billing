package testenv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	uuid "github.com/satori/go.uuid"
)

type planKey struct {
	name      string
	validFrom string
}

type spaceKey struct {
	orgGUID   string
	spaceName string
}

type appKey struct {
	orgGUID   string
	spaceGUID string
	appName   string
}

type EventInfo struct {
	CreatedAt string
	Delta     string
	State     string
	Updates   string
}

type TestScenario struct {
	plans  map[planKey]*eventio.PricingPlan
	orgs   map[string]string
	spaces map[spaceKey]string

	apps map[appKey]string

	appEvents map[appKey][]Row

	baseTime time.Time
}

func NewTestScenario(baseTimeStr string) *TestScenario {
	t, err := time.Parse("2006-01-02T15:04", baseTimeStr)
	if err != nil {
		panic(err)
	}

	scenario := TestScenario{
		plans:     map[planKey]*eventio.PricingPlan{},
		orgs:      map[string]string{},
		spaces:    map[spaceKey]string{},
		apps:      map[appKey]string{},
		appEvents: map[appKey][]Row{},
		baseTime:  t,
	}
	return &scenario
}

func (t *TestScenario) DeltaTime(ds string) time.Time {
	d, err := time.ParseDuration(ds)
	if err != nil {
		panic(err)
	}
	return t.baseTime.Add(d)
}

func (t *TestScenario) DeltaTimeRFC3339(ds string) string {
	return t.DeltaTime(ds).Format(time.RFC3339)
}

func (t *TestScenario) DeltaTimeJSON(ds string) string {
	return t.DeltaTime(ds).Format("2006-01-02T15:04:05+00:00")
}

func (t *TestScenario) Open(cfg eventstore.Config) (*TempDB, error) {
	for _, p := range t.plans {
		cfg.AddPlan(*p)
	}

	db, err := Open(cfg)
	if err != nil {
		return nil, err
	}

	t.FlushAppEvents(db)

	return db, err
}

func (t *TestScenario) FlushAppEvents(db *TempDB) error {
	for _, ev := range t.appEvents {
		err := db.Insert("app_usage_events", ev...)
		if err != nil {
			return err
		}
	}
	t.appEvents = map[appKey][]Row{}
	return nil
}

func (t *TestScenario) GetPlan(planName, validFrom string) *eventio.PricingPlan {
	k := planKey{name: planName, validFrom: validFrom}

	if _, ok := t.plans[k]; !ok {
		t.plans[k] = &eventio.PricingPlan{
			PlanGUID:   uuid.NewV4().String(),
			ValidFrom:  validFrom,
			Name:       planName,
			Components: []eventio.PricingPlanComponent{},
		}

	}

	return t.plans[k]
}

func (t *TestScenario) GetPlanGUID(planName, validFrom string) string {
	return t.GetPlan(planName, validFrom).PlanGUID
}

func (t *TestScenario) AddComponent(planName, validFrom, componentName, formula, currency, vat string) {
	p := t.GetPlan(planName, validFrom)

	p.Components = append(
		p.Components,
		eventio.PricingPlanComponent{
			Name:         componentName,
			Formula:      formula,
			CurrencyCode: currency,
			VATCode:      vat,
		},
	)
}

func (t *TestScenario) AddComputePlan() {
	k := planKey{name: "ComputePlan1", validFrom: "2001-01-01"}

	t.plans[k] = &eventio.PricingPlan{
		PlanGUID:  eventstore.ComputePlanGUID,
		ValidFrom: t.baseTime.Format("2006-01-02"),
		Name:      "ComputePlan1",
		Components: []eventio.PricingPlanComponent{
			{
				Name:         "compute",
				Formula:      "ceil($time_in_seconds/3600) * 0.01",
				CurrencyCode: "GBP",
				VATCode:      "Standard",
			},
		},
	}
}

func (t *TestScenario) GetOrgGUID(orgName string) string {
	if _, ok := t.orgs[orgName]; !ok {
		t.orgs[orgName] = uuid.NewV4().String()
	}
	return t.orgs[orgName]
}

func (t *TestScenario) GetSpaceGUID(orgName, spaceName string) string {
	orgGUID := t.GetOrgGUID(orgName)
	k := spaceKey{orgGUID: orgGUID, spaceName: spaceName}

	if _, ok := t.spaces[k]; !ok {
		t.spaces[k] = uuid.NewV4().String()
	}
	return t.spaces[k]
}

func (t *TestScenario) GetAppGUID(orgName, spaceName, appName string) string {
	orgGUID := t.GetOrgGUID(orgName)
	spaceGUID := t.GetSpaceGUID(orgName, spaceName)
	k := appKey{orgGUID: orgGUID, spaceGUID: spaceGUID, appName: appName}

	if _, ok := t.apps[k]; !ok {
		t.apps[k] = uuid.NewV4().String()
	}
	return t.apps[k]
}

func (t *TestScenario) GetAppEventGUIDs(orgName, spaceName, appName string) []string {
	orgGUID := t.GetOrgGUID(orgName)
	spaceGUID := t.GetSpaceGUID(orgName, spaceName)
	k := appKey{orgGUID: orgGUID, spaceGUID: spaceGUID, appName: appName}

	if _, ok := t.appEvents[k]; !ok {
		panic(fmt.Sprintf("No appEvents for %v", k))
	}
	guids := []string{}
	for _, r := range t.appEvents[k] {
		switch v := r["guid"].(type) {
		case string:
			guids = append(guids, v)
		default:
			panic(fmt.Sprintf(`r["guid"] is not a string, it is %T`, r["guid"]))
		}
	}
	return guids
}

func (t *TestScenario) AppLifeCycle(
	orgName string,
	spaceName string,
	appName string,
	eventsInfo ...EventInfo,
) {
	orgGUID := t.GetOrgGUID(orgName)
	spaceGUID := t.GetSpaceGUID(orgName, spaceName)
	appGUID := t.GetAppGUID(orgName, spaceName, appName)

	rawMessage := map[string]interface{}{}
	err := json.Unmarshal([]byte(`
	{
		"state" : "STOPPED",
		"process_type" : "web",
		"instance_count" : 1,
		"memory_in_mb_per_instance" : 1024
	}
	`), &rawMessage)
	if err != nil {
		panic(err)
	}

	rawMessage["app_name"] = appName
	rawMessage["app_guid"] = appGUID
	rawMessage["org_guid"] = orgGUID
	rawMessage["space_guid"] = spaceGUID
	rawMessage["space_name"] = spaceName

	k := appKey{orgGUID: orgGUID, spaceGUID: spaceGUID, appName: appName}
	t.appEvents[k] = []Row{}
	for _, e := range eventsInfo {
		rawMessage["previous_state"] = rawMessage["state"]
		rawMessage["state"] = e.State

		if e.Updates != "" {
			rawMessageUpdates := map[string]interface{}{}
			err := json.Unmarshal([]byte(e.Updates), &rawMessageUpdates)
			if err != nil {
				panic(err)
			}
			for k, v := range rawMessageUpdates {
				rawMessage[k] = v
			}
		}

		rawMessageBytes, err := json.Marshal(&rawMessage)
		if err != nil {
			panic(err)
		}

		createdAt := e.CreatedAt
		if createdAt == "" {
			delta, err := time.ParseDuration(e.Delta)
			if err != nil {
				panic("Invalid event.createdAt or event.Delta")
			}
			createdAt = t.baseTime.Add(delta).Format(time.RFC3339)
		}
		event := Row{
			"guid":        uuid.NewV4().String(),
			"created_at":  createdAt,
			"raw_message": json.RawMessage(rawMessageBytes),
		}

		t.appEvents[k] = append(t.appEvents[k], event)
	}
}
