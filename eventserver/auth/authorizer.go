package auth

type Authorizer interface {
	Admin() (bool, error)
	Organisations([]string) (bool, error)
}
