package auth

type Authorizer interface {
	Spaces() ([]string, error)
	Admin() (bool, error)
}
