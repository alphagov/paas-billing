package auth

//counterfeiter:generate . Authorizer
type Authorizer interface {
	Admin() (bool, error)
	HasBillingAccess([]string) (bool, error)
}
