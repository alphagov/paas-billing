// Code generated by counterfeiter. DO NOT EDIT.
package instancediscovererfakes

import (
	"net/url"
	"sync"

	"github.com/alphagov/paas-billing/instancediscoverer"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type FakeCFAppDiscoverer struct {
	GetAppRouteURLsByNameStub        func(string) ([]*url.URL, error)
	getAppRouteURLsByNameMutex       sync.RWMutex
	getAppRouteURLsByNameArgsForCall []struct {
		arg1 string
	}
	getAppRouteURLsByNameReturns struct {
		result1 []*url.URL
		result2 error
	}
	getAppRouteURLsByNameReturnsOnCall map[int]struct {
		result1 []*url.URL
		result2 error
	}
	GetAppRoutesByNameStub        func(string) ([]cfclient.Route, error)
	getAppRoutesByNameMutex       sync.RWMutex
	getAppRoutesByNameArgsForCall []struct {
		arg1 string
	}
	getAppRoutesByNameReturns struct {
		result1 []cfclient.Route
		result2 error
	}
	getAppRoutesByNameReturnsOnCall map[int]struct {
		result1 []cfclient.Route
		result2 error
	}
	GetSpaceAppByNameStub        func(string) (cfclient.App, error)
	getSpaceAppByNameMutex       sync.RWMutex
	getSpaceAppByNameArgsForCall []struct {
		arg1 string
	}
	getSpaceAppByNameReturns struct {
		result1 cfclient.App
		result2 error
	}
	getSpaceAppByNameReturnsOnCall map[int]struct {
		result1 cfclient.App
		result2 error
	}
	MyNameStub        func() string
	myNameMutex       sync.RWMutex
	myNameArgsForCall []struct {
	}
	myNameReturns struct {
		result1 string
	}
	myNameReturnsOnCall map[int]struct {
		result1 string
	}
	OrgStub        func() cfclient.Org
	orgMutex       sync.RWMutex
	orgArgsForCall []struct {
	}
	orgReturns struct {
		result1 cfclient.Org
	}
	orgReturnsOnCall map[int]struct {
		result1 cfclient.Org
	}
	PingStub        func() error
	pingMutex       sync.RWMutex
	pingArgsForCall []struct {
	}
	pingReturns struct {
		result1 error
	}
	pingReturnsOnCall map[int]struct {
		result1 error
	}
	SpaceStub        func() cfclient.Space
	spaceMutex       sync.RWMutex
	spaceArgsForCall []struct {
	}
	spaceReturns struct {
		result1 cfclient.Space
	}
	spaceReturnsOnCall map[int]struct {
		result1 cfclient.Space
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCFAppDiscoverer) GetAppRouteURLsByName(arg1 string) ([]*url.URL, error) {
	fake.getAppRouteURLsByNameMutex.Lock()
	ret, specificReturn := fake.getAppRouteURLsByNameReturnsOnCall[len(fake.getAppRouteURLsByNameArgsForCall)]
	fake.getAppRouteURLsByNameArgsForCall = append(fake.getAppRouteURLsByNameArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetAppRouteURLsByNameStub
	fakeReturns := fake.getAppRouteURLsByNameReturns
	fake.recordInvocation("GetAppRouteURLsByName", []interface{}{arg1})
	fake.getAppRouteURLsByNameMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFAppDiscoverer) GetAppRouteURLsByNameCallCount() int {
	fake.getAppRouteURLsByNameMutex.RLock()
	defer fake.getAppRouteURLsByNameMutex.RUnlock()
	return len(fake.getAppRouteURLsByNameArgsForCall)
}

func (fake *FakeCFAppDiscoverer) GetAppRouteURLsByNameCalls(stub func(string) ([]*url.URL, error)) {
	fake.getAppRouteURLsByNameMutex.Lock()
	defer fake.getAppRouteURLsByNameMutex.Unlock()
	fake.GetAppRouteURLsByNameStub = stub
}

func (fake *FakeCFAppDiscoverer) GetAppRouteURLsByNameArgsForCall(i int) string {
	fake.getAppRouteURLsByNameMutex.RLock()
	defer fake.getAppRouteURLsByNameMutex.RUnlock()
	argsForCall := fake.getAppRouteURLsByNameArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeCFAppDiscoverer) GetAppRouteURLsByNameReturns(result1 []*url.URL, result2 error) {
	fake.getAppRouteURLsByNameMutex.Lock()
	defer fake.getAppRouteURLsByNameMutex.Unlock()
	fake.GetAppRouteURLsByNameStub = nil
	fake.getAppRouteURLsByNameReturns = struct {
		result1 []*url.URL
		result2 error
	}{result1, result2}
}

func (fake *FakeCFAppDiscoverer) GetAppRouteURLsByNameReturnsOnCall(i int, result1 []*url.URL, result2 error) {
	fake.getAppRouteURLsByNameMutex.Lock()
	defer fake.getAppRouteURLsByNameMutex.Unlock()
	fake.GetAppRouteURLsByNameStub = nil
	if fake.getAppRouteURLsByNameReturnsOnCall == nil {
		fake.getAppRouteURLsByNameReturnsOnCall = make(map[int]struct {
			result1 []*url.URL
			result2 error
		})
	}
	fake.getAppRouteURLsByNameReturnsOnCall[i] = struct {
		result1 []*url.URL
		result2 error
	}{result1, result2}
}

func (fake *FakeCFAppDiscoverer) GetAppRoutesByName(arg1 string) ([]cfclient.Route, error) {
	fake.getAppRoutesByNameMutex.Lock()
	ret, specificReturn := fake.getAppRoutesByNameReturnsOnCall[len(fake.getAppRoutesByNameArgsForCall)]
	fake.getAppRoutesByNameArgsForCall = append(fake.getAppRoutesByNameArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetAppRoutesByNameStub
	fakeReturns := fake.getAppRoutesByNameReturns
	fake.recordInvocation("GetAppRoutesByName", []interface{}{arg1})
	fake.getAppRoutesByNameMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFAppDiscoverer) GetAppRoutesByNameCallCount() int {
	fake.getAppRoutesByNameMutex.RLock()
	defer fake.getAppRoutesByNameMutex.RUnlock()
	return len(fake.getAppRoutesByNameArgsForCall)
}

func (fake *FakeCFAppDiscoverer) GetAppRoutesByNameCalls(stub func(string) ([]cfclient.Route, error)) {
	fake.getAppRoutesByNameMutex.Lock()
	defer fake.getAppRoutesByNameMutex.Unlock()
	fake.GetAppRoutesByNameStub = stub
}

func (fake *FakeCFAppDiscoverer) GetAppRoutesByNameArgsForCall(i int) string {
	fake.getAppRoutesByNameMutex.RLock()
	defer fake.getAppRoutesByNameMutex.RUnlock()
	argsForCall := fake.getAppRoutesByNameArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeCFAppDiscoverer) GetAppRoutesByNameReturns(result1 []cfclient.Route, result2 error) {
	fake.getAppRoutesByNameMutex.Lock()
	defer fake.getAppRoutesByNameMutex.Unlock()
	fake.GetAppRoutesByNameStub = nil
	fake.getAppRoutesByNameReturns = struct {
		result1 []cfclient.Route
		result2 error
	}{result1, result2}
}

func (fake *FakeCFAppDiscoverer) GetAppRoutesByNameReturnsOnCall(i int, result1 []cfclient.Route, result2 error) {
	fake.getAppRoutesByNameMutex.Lock()
	defer fake.getAppRoutesByNameMutex.Unlock()
	fake.GetAppRoutesByNameStub = nil
	if fake.getAppRoutesByNameReturnsOnCall == nil {
		fake.getAppRoutesByNameReturnsOnCall = make(map[int]struct {
			result1 []cfclient.Route
			result2 error
		})
	}
	fake.getAppRoutesByNameReturnsOnCall[i] = struct {
		result1 []cfclient.Route
		result2 error
	}{result1, result2}
}

func (fake *FakeCFAppDiscoverer) GetSpaceAppByName(arg1 string) (cfclient.App, error) {
	fake.getSpaceAppByNameMutex.Lock()
	ret, specificReturn := fake.getSpaceAppByNameReturnsOnCall[len(fake.getSpaceAppByNameArgsForCall)]
	fake.getSpaceAppByNameArgsForCall = append(fake.getSpaceAppByNameArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetSpaceAppByNameStub
	fakeReturns := fake.getSpaceAppByNameReturns
	fake.recordInvocation("GetSpaceAppByName", []interface{}{arg1})
	fake.getSpaceAppByNameMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFAppDiscoverer) GetSpaceAppByNameCallCount() int {
	fake.getSpaceAppByNameMutex.RLock()
	defer fake.getSpaceAppByNameMutex.RUnlock()
	return len(fake.getSpaceAppByNameArgsForCall)
}

func (fake *FakeCFAppDiscoverer) GetSpaceAppByNameCalls(stub func(string) (cfclient.App, error)) {
	fake.getSpaceAppByNameMutex.Lock()
	defer fake.getSpaceAppByNameMutex.Unlock()
	fake.GetSpaceAppByNameStub = stub
}

func (fake *FakeCFAppDiscoverer) GetSpaceAppByNameArgsForCall(i int) string {
	fake.getSpaceAppByNameMutex.RLock()
	defer fake.getSpaceAppByNameMutex.RUnlock()
	argsForCall := fake.getSpaceAppByNameArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeCFAppDiscoverer) GetSpaceAppByNameReturns(result1 cfclient.App, result2 error) {
	fake.getSpaceAppByNameMutex.Lock()
	defer fake.getSpaceAppByNameMutex.Unlock()
	fake.GetSpaceAppByNameStub = nil
	fake.getSpaceAppByNameReturns = struct {
		result1 cfclient.App
		result2 error
	}{result1, result2}
}

func (fake *FakeCFAppDiscoverer) GetSpaceAppByNameReturnsOnCall(i int, result1 cfclient.App, result2 error) {
	fake.getSpaceAppByNameMutex.Lock()
	defer fake.getSpaceAppByNameMutex.Unlock()
	fake.GetSpaceAppByNameStub = nil
	if fake.getSpaceAppByNameReturnsOnCall == nil {
		fake.getSpaceAppByNameReturnsOnCall = make(map[int]struct {
			result1 cfclient.App
			result2 error
		})
	}
	fake.getSpaceAppByNameReturnsOnCall[i] = struct {
		result1 cfclient.App
		result2 error
	}{result1, result2}
}

func (fake *FakeCFAppDiscoverer) MyName() string {
	fake.myNameMutex.Lock()
	ret, specificReturn := fake.myNameReturnsOnCall[len(fake.myNameArgsForCall)]
	fake.myNameArgsForCall = append(fake.myNameArgsForCall, struct {
	}{})
	stub := fake.MyNameStub
	fakeReturns := fake.myNameReturns
	fake.recordInvocation("MyName", []interface{}{})
	fake.myNameMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeCFAppDiscoverer) MyNameCallCount() int {
	fake.myNameMutex.RLock()
	defer fake.myNameMutex.RUnlock()
	return len(fake.myNameArgsForCall)
}

func (fake *FakeCFAppDiscoverer) MyNameCalls(stub func() string) {
	fake.myNameMutex.Lock()
	defer fake.myNameMutex.Unlock()
	fake.MyNameStub = stub
}

func (fake *FakeCFAppDiscoverer) MyNameReturns(result1 string) {
	fake.myNameMutex.Lock()
	defer fake.myNameMutex.Unlock()
	fake.MyNameStub = nil
	fake.myNameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeCFAppDiscoverer) MyNameReturnsOnCall(i int, result1 string) {
	fake.myNameMutex.Lock()
	defer fake.myNameMutex.Unlock()
	fake.MyNameStub = nil
	if fake.myNameReturnsOnCall == nil {
		fake.myNameReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.myNameReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeCFAppDiscoverer) Org() cfclient.Org {
	fake.orgMutex.Lock()
	ret, specificReturn := fake.orgReturnsOnCall[len(fake.orgArgsForCall)]
	fake.orgArgsForCall = append(fake.orgArgsForCall, struct {
	}{})
	stub := fake.OrgStub
	fakeReturns := fake.orgReturns
	fake.recordInvocation("Org", []interface{}{})
	fake.orgMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeCFAppDiscoverer) OrgCallCount() int {
	fake.orgMutex.RLock()
	defer fake.orgMutex.RUnlock()
	return len(fake.orgArgsForCall)
}

func (fake *FakeCFAppDiscoverer) OrgCalls(stub func() cfclient.Org) {
	fake.orgMutex.Lock()
	defer fake.orgMutex.Unlock()
	fake.OrgStub = stub
}

func (fake *FakeCFAppDiscoverer) OrgReturns(result1 cfclient.Org) {
	fake.orgMutex.Lock()
	defer fake.orgMutex.Unlock()
	fake.OrgStub = nil
	fake.orgReturns = struct {
		result1 cfclient.Org
	}{result1}
}

func (fake *FakeCFAppDiscoverer) OrgReturnsOnCall(i int, result1 cfclient.Org) {
	fake.orgMutex.Lock()
	defer fake.orgMutex.Unlock()
	fake.OrgStub = nil
	if fake.orgReturnsOnCall == nil {
		fake.orgReturnsOnCall = make(map[int]struct {
			result1 cfclient.Org
		})
	}
	fake.orgReturnsOnCall[i] = struct {
		result1 cfclient.Org
	}{result1}
}

func (fake *FakeCFAppDiscoverer) Ping() error {
	fake.pingMutex.Lock()
	ret, specificReturn := fake.pingReturnsOnCall[len(fake.pingArgsForCall)]
	fake.pingArgsForCall = append(fake.pingArgsForCall, struct {
	}{})
	stub := fake.PingStub
	fakeReturns := fake.pingReturns
	fake.recordInvocation("Ping", []interface{}{})
	fake.pingMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeCFAppDiscoverer) PingCallCount() int {
	fake.pingMutex.RLock()
	defer fake.pingMutex.RUnlock()
	return len(fake.pingArgsForCall)
}

func (fake *FakeCFAppDiscoverer) PingCalls(stub func() error) {
	fake.pingMutex.Lock()
	defer fake.pingMutex.Unlock()
	fake.PingStub = stub
}

func (fake *FakeCFAppDiscoverer) PingReturns(result1 error) {
	fake.pingMutex.Lock()
	defer fake.pingMutex.Unlock()
	fake.PingStub = nil
	fake.pingReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCFAppDiscoverer) PingReturnsOnCall(i int, result1 error) {
	fake.pingMutex.Lock()
	defer fake.pingMutex.Unlock()
	fake.PingStub = nil
	if fake.pingReturnsOnCall == nil {
		fake.pingReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.pingReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCFAppDiscoverer) Space() cfclient.Space {
	fake.spaceMutex.Lock()
	ret, specificReturn := fake.spaceReturnsOnCall[len(fake.spaceArgsForCall)]
	fake.spaceArgsForCall = append(fake.spaceArgsForCall, struct {
	}{})
	stub := fake.SpaceStub
	fakeReturns := fake.spaceReturns
	fake.recordInvocation("Space", []interface{}{})
	fake.spaceMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeCFAppDiscoverer) SpaceCallCount() int {
	fake.spaceMutex.RLock()
	defer fake.spaceMutex.RUnlock()
	return len(fake.spaceArgsForCall)
}

func (fake *FakeCFAppDiscoverer) SpaceCalls(stub func() cfclient.Space) {
	fake.spaceMutex.Lock()
	defer fake.spaceMutex.Unlock()
	fake.SpaceStub = stub
}

func (fake *FakeCFAppDiscoverer) SpaceReturns(result1 cfclient.Space) {
	fake.spaceMutex.Lock()
	defer fake.spaceMutex.Unlock()
	fake.SpaceStub = nil
	fake.spaceReturns = struct {
		result1 cfclient.Space
	}{result1}
}

func (fake *FakeCFAppDiscoverer) SpaceReturnsOnCall(i int, result1 cfclient.Space) {
	fake.spaceMutex.Lock()
	defer fake.spaceMutex.Unlock()
	fake.SpaceStub = nil
	if fake.spaceReturnsOnCall == nil {
		fake.spaceReturnsOnCall = make(map[int]struct {
			result1 cfclient.Space
		})
	}
	fake.spaceReturnsOnCall[i] = struct {
		result1 cfclient.Space
	}{result1}
}

func (fake *FakeCFAppDiscoverer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getAppRouteURLsByNameMutex.RLock()
	defer fake.getAppRouteURLsByNameMutex.RUnlock()
	fake.getAppRoutesByNameMutex.RLock()
	defer fake.getAppRoutesByNameMutex.RUnlock()
	fake.getSpaceAppByNameMutex.RLock()
	defer fake.getSpaceAppByNameMutex.RUnlock()
	fake.myNameMutex.RLock()
	defer fake.myNameMutex.RUnlock()
	fake.orgMutex.RLock()
	defer fake.orgMutex.RUnlock()
	fake.pingMutex.RLock()
	defer fake.pingMutex.RUnlock()
	fake.spaceMutex.RLock()
	defer fake.spaceMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeCFAppDiscoverer) recordInvocation(key string, args []interface{}) {
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

var _ instancediscoverer.CFAppDiscoverer = new(FakeCFAppDiscoverer)
