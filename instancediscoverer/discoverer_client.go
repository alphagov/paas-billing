package instancediscoverer

import "github.com/cloudfoundry-community/go-cfclient"

//counterfeiter:generate . DiscovererClient
type DiscovererClient interface {
	AppByName(appName, spaceGuid, orgGuid string) (cfclient.App, error)
	GetAppRoutes(guid string) ([]cfclient.Route, error)
	GetOrgByGuid(guid string) (cfclient.Org, error)
	GetSpaceByGuid(guid string) (cfclient.Space, error)
	GetDomainByGuid(guid string) (cfclient.Domain, error)
}
