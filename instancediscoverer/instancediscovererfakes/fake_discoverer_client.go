// Code generated by counterfeiter. DO NOT EDIT.
package instancediscovererfakes

import (
	"sync"

	"github.com/alphagov/paas-billing/instancediscoverer"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type FakeDiscovererClient struct {
	AppByNameStub        func(string, string, string) (cfclient.App, error)
	appByNameMutex       sync.RWMutex
	appByNameArgsForCall []struct {
		arg1 string
		arg2 string
		arg3 string
	}
	appByNameReturns struct {
		result1 cfclient.App
		result2 error
	}
	appByNameReturnsOnCall map[int]struct {
		result1 cfclient.App
		result2 error
	}
	GetAppRoutesStub        func(string) ([]cfclient.Route, error)
	getAppRoutesMutex       sync.RWMutex
	getAppRoutesArgsForCall []struct {
		arg1 string
	}
	getAppRoutesReturns struct {
		result1 []cfclient.Route
		result2 error
	}
	getAppRoutesReturnsOnCall map[int]struct {
		result1 []cfclient.Route
		result2 error
	}
	GetDomainByGuidStub        func(string) (cfclient.Domain, error)
	getDomainByGuidMutex       sync.RWMutex
	getDomainByGuidArgsForCall []struct {
		arg1 string
	}
	getDomainByGuidReturns struct {
		result1 cfclient.Domain
		result2 error
	}
	getDomainByGuidReturnsOnCall map[int]struct {
		result1 cfclient.Domain
		result2 error
	}
	GetOrgByGuidStub        func(string) (cfclient.Org, error)
	getOrgByGuidMutex       sync.RWMutex
	getOrgByGuidArgsForCall []struct {
		arg1 string
	}
	getOrgByGuidReturns struct {
		result1 cfclient.Org
		result2 error
	}
	getOrgByGuidReturnsOnCall map[int]struct {
		result1 cfclient.Org
		result2 error
	}
	GetSpaceByGuidStub        func(string) (cfclient.Space, error)
	getSpaceByGuidMutex       sync.RWMutex
	getSpaceByGuidArgsForCall []struct {
		arg1 string
	}
	getSpaceByGuidReturns struct {
		result1 cfclient.Space
		result2 error
	}
	getSpaceByGuidReturnsOnCall map[int]struct {
		result1 cfclient.Space
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDiscovererClient) AppByName(arg1 string, arg2 string, arg3 string) (cfclient.App, error) {
	fake.appByNameMutex.Lock()
	ret, specificReturn := fake.appByNameReturnsOnCall[len(fake.appByNameArgsForCall)]
	fake.appByNameArgsForCall = append(fake.appByNameArgsForCall, struct {
		arg1 string
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.AppByNameStub
	fakeReturns := fake.appByNameReturns
	fake.recordInvocation("AppByName", []interface{}{arg1, arg2, arg3})
	fake.appByNameMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDiscovererClient) AppByNameCallCount() int {
	fake.appByNameMutex.RLock()
	defer fake.appByNameMutex.RUnlock()
	return len(fake.appByNameArgsForCall)
}

func (fake *FakeDiscovererClient) AppByNameCalls(stub func(string, string, string) (cfclient.App, error)) {
	fake.appByNameMutex.Lock()
	defer fake.appByNameMutex.Unlock()
	fake.AppByNameStub = stub
}

func (fake *FakeDiscovererClient) AppByNameArgsForCall(i int) (string, string, string) {
	fake.appByNameMutex.RLock()
	defer fake.appByNameMutex.RUnlock()
	argsForCall := fake.appByNameArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeDiscovererClient) AppByNameReturns(result1 cfclient.App, result2 error) {
	fake.appByNameMutex.Lock()
	defer fake.appByNameMutex.Unlock()
	fake.AppByNameStub = nil
	fake.appByNameReturns = struct {
		result1 cfclient.App
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) AppByNameReturnsOnCall(i int, result1 cfclient.App, result2 error) {
	fake.appByNameMutex.Lock()
	defer fake.appByNameMutex.Unlock()
	fake.AppByNameStub = nil
	if fake.appByNameReturnsOnCall == nil {
		fake.appByNameReturnsOnCall = make(map[int]struct {
			result1 cfclient.App
			result2 error
		})
	}
	fake.appByNameReturnsOnCall[i] = struct {
		result1 cfclient.App
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetAppRoutes(arg1 string) ([]cfclient.Route, error) {
	fake.getAppRoutesMutex.Lock()
	ret, specificReturn := fake.getAppRoutesReturnsOnCall[len(fake.getAppRoutesArgsForCall)]
	fake.getAppRoutesArgsForCall = append(fake.getAppRoutesArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetAppRoutesStub
	fakeReturns := fake.getAppRoutesReturns
	fake.recordInvocation("GetAppRoutes", []interface{}{arg1})
	fake.getAppRoutesMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDiscovererClient) GetAppRoutesCallCount() int {
	fake.getAppRoutesMutex.RLock()
	defer fake.getAppRoutesMutex.RUnlock()
	return len(fake.getAppRoutesArgsForCall)
}

func (fake *FakeDiscovererClient) GetAppRoutesCalls(stub func(string) ([]cfclient.Route, error)) {
	fake.getAppRoutesMutex.Lock()
	defer fake.getAppRoutesMutex.Unlock()
	fake.GetAppRoutesStub = stub
}

func (fake *FakeDiscovererClient) GetAppRoutesArgsForCall(i int) string {
	fake.getAppRoutesMutex.RLock()
	defer fake.getAppRoutesMutex.RUnlock()
	argsForCall := fake.getAppRoutesArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDiscovererClient) GetAppRoutesReturns(result1 []cfclient.Route, result2 error) {
	fake.getAppRoutesMutex.Lock()
	defer fake.getAppRoutesMutex.Unlock()
	fake.GetAppRoutesStub = nil
	fake.getAppRoutesReturns = struct {
		result1 []cfclient.Route
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetAppRoutesReturnsOnCall(i int, result1 []cfclient.Route, result2 error) {
	fake.getAppRoutesMutex.Lock()
	defer fake.getAppRoutesMutex.Unlock()
	fake.GetAppRoutesStub = nil
	if fake.getAppRoutesReturnsOnCall == nil {
		fake.getAppRoutesReturnsOnCall = make(map[int]struct {
			result1 []cfclient.Route
			result2 error
		})
	}
	fake.getAppRoutesReturnsOnCall[i] = struct {
		result1 []cfclient.Route
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetDomainByGuid(arg1 string) (cfclient.Domain, error) {
	fake.getDomainByGuidMutex.Lock()
	ret, specificReturn := fake.getDomainByGuidReturnsOnCall[len(fake.getDomainByGuidArgsForCall)]
	fake.getDomainByGuidArgsForCall = append(fake.getDomainByGuidArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetDomainByGuidStub
	fakeReturns := fake.getDomainByGuidReturns
	fake.recordInvocation("GetDomainByGuid", []interface{}{arg1})
	fake.getDomainByGuidMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDiscovererClient) GetDomainByGuidCallCount() int {
	fake.getDomainByGuidMutex.RLock()
	defer fake.getDomainByGuidMutex.RUnlock()
	return len(fake.getDomainByGuidArgsForCall)
}

func (fake *FakeDiscovererClient) GetDomainByGuidCalls(stub func(string) (cfclient.Domain, error)) {
	fake.getDomainByGuidMutex.Lock()
	defer fake.getDomainByGuidMutex.Unlock()
	fake.GetDomainByGuidStub = stub
}

func (fake *FakeDiscovererClient) GetDomainByGuidArgsForCall(i int) string {
	fake.getDomainByGuidMutex.RLock()
	defer fake.getDomainByGuidMutex.RUnlock()
	argsForCall := fake.getDomainByGuidArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDiscovererClient) GetDomainByGuidReturns(result1 cfclient.Domain, result2 error) {
	fake.getDomainByGuidMutex.Lock()
	defer fake.getDomainByGuidMutex.Unlock()
	fake.GetDomainByGuidStub = nil
	fake.getDomainByGuidReturns = struct {
		result1 cfclient.Domain
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetDomainByGuidReturnsOnCall(i int, result1 cfclient.Domain, result2 error) {
	fake.getDomainByGuidMutex.Lock()
	defer fake.getDomainByGuidMutex.Unlock()
	fake.GetDomainByGuidStub = nil
	if fake.getDomainByGuidReturnsOnCall == nil {
		fake.getDomainByGuidReturnsOnCall = make(map[int]struct {
			result1 cfclient.Domain
			result2 error
		})
	}
	fake.getDomainByGuidReturnsOnCall[i] = struct {
		result1 cfclient.Domain
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetOrgByGuid(arg1 string) (cfclient.Org, error) {
	fake.getOrgByGuidMutex.Lock()
	ret, specificReturn := fake.getOrgByGuidReturnsOnCall[len(fake.getOrgByGuidArgsForCall)]
	fake.getOrgByGuidArgsForCall = append(fake.getOrgByGuidArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetOrgByGuidStub
	fakeReturns := fake.getOrgByGuidReturns
	fake.recordInvocation("GetOrgByGuid", []interface{}{arg1})
	fake.getOrgByGuidMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDiscovererClient) GetOrgByGuidCallCount() int {
	fake.getOrgByGuidMutex.RLock()
	defer fake.getOrgByGuidMutex.RUnlock()
	return len(fake.getOrgByGuidArgsForCall)
}

func (fake *FakeDiscovererClient) GetOrgByGuidCalls(stub func(string) (cfclient.Org, error)) {
	fake.getOrgByGuidMutex.Lock()
	defer fake.getOrgByGuidMutex.Unlock()
	fake.GetOrgByGuidStub = stub
}

func (fake *FakeDiscovererClient) GetOrgByGuidArgsForCall(i int) string {
	fake.getOrgByGuidMutex.RLock()
	defer fake.getOrgByGuidMutex.RUnlock()
	argsForCall := fake.getOrgByGuidArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDiscovererClient) GetOrgByGuidReturns(result1 cfclient.Org, result2 error) {
	fake.getOrgByGuidMutex.Lock()
	defer fake.getOrgByGuidMutex.Unlock()
	fake.GetOrgByGuidStub = nil
	fake.getOrgByGuidReturns = struct {
		result1 cfclient.Org
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetOrgByGuidReturnsOnCall(i int, result1 cfclient.Org, result2 error) {
	fake.getOrgByGuidMutex.Lock()
	defer fake.getOrgByGuidMutex.Unlock()
	fake.GetOrgByGuidStub = nil
	if fake.getOrgByGuidReturnsOnCall == nil {
		fake.getOrgByGuidReturnsOnCall = make(map[int]struct {
			result1 cfclient.Org
			result2 error
		})
	}
	fake.getOrgByGuidReturnsOnCall[i] = struct {
		result1 cfclient.Org
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetSpaceByGuid(arg1 string) (cfclient.Space, error) {
	fake.getSpaceByGuidMutex.Lock()
	ret, specificReturn := fake.getSpaceByGuidReturnsOnCall[len(fake.getSpaceByGuidArgsForCall)]
	fake.getSpaceByGuidArgsForCall = append(fake.getSpaceByGuidArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetSpaceByGuidStub
	fakeReturns := fake.getSpaceByGuidReturns
	fake.recordInvocation("GetSpaceByGuid", []interface{}{arg1})
	fake.getSpaceByGuidMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDiscovererClient) GetSpaceByGuidCallCount() int {
	fake.getSpaceByGuidMutex.RLock()
	defer fake.getSpaceByGuidMutex.RUnlock()
	return len(fake.getSpaceByGuidArgsForCall)
}

func (fake *FakeDiscovererClient) GetSpaceByGuidCalls(stub func(string) (cfclient.Space, error)) {
	fake.getSpaceByGuidMutex.Lock()
	defer fake.getSpaceByGuidMutex.Unlock()
	fake.GetSpaceByGuidStub = stub
}

func (fake *FakeDiscovererClient) GetSpaceByGuidArgsForCall(i int) string {
	fake.getSpaceByGuidMutex.RLock()
	defer fake.getSpaceByGuidMutex.RUnlock()
	argsForCall := fake.getSpaceByGuidArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDiscovererClient) GetSpaceByGuidReturns(result1 cfclient.Space, result2 error) {
	fake.getSpaceByGuidMutex.Lock()
	defer fake.getSpaceByGuidMutex.Unlock()
	fake.GetSpaceByGuidStub = nil
	fake.getSpaceByGuidReturns = struct {
		result1 cfclient.Space
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) GetSpaceByGuidReturnsOnCall(i int, result1 cfclient.Space, result2 error) {
	fake.getSpaceByGuidMutex.Lock()
	defer fake.getSpaceByGuidMutex.Unlock()
	fake.GetSpaceByGuidStub = nil
	if fake.getSpaceByGuidReturnsOnCall == nil {
		fake.getSpaceByGuidReturnsOnCall = make(map[int]struct {
			result1 cfclient.Space
			result2 error
		})
	}
	fake.getSpaceByGuidReturnsOnCall[i] = struct {
		result1 cfclient.Space
		result2 error
	}{result1, result2}
}

func (fake *FakeDiscovererClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.appByNameMutex.RLock()
	defer fake.appByNameMutex.RUnlock()
	fake.getAppRoutesMutex.RLock()
	defer fake.getAppRoutesMutex.RUnlock()
	fake.getDomainByGuidMutex.RLock()
	defer fake.getDomainByGuidMutex.RUnlock()
	fake.getOrgByGuidMutex.RLock()
	defer fake.getOrgByGuidMutex.RUnlock()
	fake.getSpaceByGuidMutex.RLock()
	defer fake.getSpaceByGuidMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDiscovererClient) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ instancediscoverer.DiscovererClient = new(FakeDiscovererClient)
