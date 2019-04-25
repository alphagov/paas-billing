package auth

type Authorizer interface {
	Admin() (bool, error)
	HasBillingAccess([]string) (bool, error)
}
