package instancediscoverer

import (
	"code.cloudfoundry.org/lager"
	"errors"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"net/url"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . CFAppDiscoverer
type CFAppDiscoverer interface {
	GetSpaceAppByName(appName string) (cfclient.App, error)
	GetAppRoutesByName(appName string) ([]cfclient.Route, error)
	GetAppRouteURLsByName(appName string) ([]*url.URL, error)
	Org() cfclient.Org
	Space() cfclient.Space
	MyName() string
	Ping() error
}

type cfAppDiscoverer struct {
	DiscoveryScope AppDiscoveryScope
	client         DiscovererClient
	logger         lager.Logger
	org            cfclient.Org
	space          cfclient.Space
	thisAppName    string
}

type Config struct {
	ClientConfig   *cfclient.Config
	Client         DiscovererClient
	Logger         lager.Logger
	DiscoveryScope AppDiscoveryScope
	ThisAppName    string
}

type AppDiscoveryScope struct {
	SpaceName        string
	SpaceID          string
	OrganizationName string
	OrganizationID   string
	AppNames         []string
}

func New(cfg Config) (CFAppDiscoverer, error) {
	if cfg.Logger == nil {
		cfg.Logger = lager.NewLogger("app-discovery")
	}
	if cfg.Client == nil {
		if cfg.ClientConfig == nil {
			return nil, fmt.Errorf("appdiscoverer.New: must supply cfclient.Config")
		}
		cf, err := cfclient.NewClient(cfg.ClientConfig)
		if err != nil {
			return nil, err
		}

		cfg.Client = cf
	}
	org, err := cfg.Client.GetOrgByGuid(cfg.DiscoveryScope.OrganizationID)
	if err != nil {
		return nil, err
	}
	space, err := cfg.Client.GetSpaceByGuid(cfg.DiscoveryScope.SpaceID)
	if err != nil {
		return nil, err
	}

	discoverer := &cfAppDiscoverer{
		org:            org,
		space:          space,
		client:         cfg.Client,
		logger:         cfg.Logger,
		DiscoveryScope: cfg.DiscoveryScope,
		thisAppName:    cfg.ThisAppName,
	}
	return discoverer, nil
}

var AccessDeniedError = errors.New("access denied")

func (d *cfAppDiscoverer) GetSpaceAppByName(appName string) (cfclient.App, error) {
	if !d.appAccessAllowed(appName) {
		return cfclient.App{}, AccessDeniedError
	}
	app, err := d.client.AppByName(appName, d.space.Guid, d.org.Guid)
	if err != nil {
		return cfclient.App{}, err
	}
	return app, nil
}

func (d *cfAppDiscoverer) GetAppRoutesByName(appName string) ([]cfclient.Route, error) {
	app, err := d.GetSpaceAppByName(appName)
	if err != nil {
		return nil, err
	}
	routes, err := d.client.GetAppRoutes(app.Guid)
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func (d *cfAppDiscoverer) GetAppRouteURLsByName(appName string) ([]*url.URL, error) {
	appRoutes, err := d.GetAppRoutesByName(appName)
	var appURLs []*url.URL
	if err != nil {
		return nil, err
	}
	for _, route := range appRoutes {
		domain, err := d.getRouteDomain(route)
		if err != nil {
			continue
		}
		routeURL := &url.URL{
			Host:   fmt.Sprintf("%s.%s", route.Host, domain.Name),
			Scheme: "https",
		}
		appURLs = append(appURLs, routeURL)
	}
	return appURLs, nil
}

func (d *cfAppDiscoverer) getRouteDomain(route cfclient.Route) (cfclient.Domain, error) {
	return d.client.GetDomainByGuid(route.DomainGuid)
}

func (d *cfAppDiscoverer) Org() cfclient.Org {
	return d.org
}

func (d *cfAppDiscoverer) Space() cfclient.Space {
	return d.space
}

func (d *cfAppDiscoverer) MyName() string {
	return d.thisAppName
}

func (d *cfAppDiscoverer) Ping() error {
	app, err := d.client.AppByName(d.thisAppName, d.space.Guid, d.org.Guid)
	if err != nil {
		return err
	}
	if app.Name != d.thisAppName {
		return fmt.Errorf("app name did not match")
	}
	return nil
}

func (d *cfAppDiscoverer) appAccessAllowed(appName string) bool {
	for _, name := range d.DiscoveryScope.AppNames {
		if name == appName {
			return true
		}
	}
	return false
}
