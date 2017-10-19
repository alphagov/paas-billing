package cfclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

//Client used to communicate with Cloud Foundry
type Client struct {
	Config   Config
	Endpoint Endpoint
}

type Endpoint struct {
	DopplerEndpoint string `json:"doppler_logging_endpoint"`
	LoggingEndpoint string `json:"logging_endpoint"`
	AuthEndpoint    string `json:"authorization_endpoint"`
	TokenEndpoint   string `json:"token_endpoint"`
}

//Config is used to configure the creation of a client
type Config struct {
	ApiAddress        string `json:"api_url"`
	Username          string `json:"user"`
	Password          string `json:"password"`
	ClientID          string `json:"client_id"`
	ClientSecret      string `json:"client_secret"`
	SkipSslValidation bool   `json:"skip_ssl_validation"`
	HttpClient        *http.Client
	Token             string `json:"auth_token"`
	TokenSource       oauth2.TokenSource
	UserAgent         string `json:"user_agent"`
}

// request is used to help build up a request
type request struct {
	method string
	url    string
	params url.Values
	body   io.Reader
	obj    interface{}
}

//DefaultConfig configuration for client
//Keep LoginAdress for backward compatibility
//Need to be remove in close future
func DefaultConfig() *Config {
	return &Config{
		ApiAddress:        "http://api.bosh-lite.com",
		Username:          "admin",
		Password:          "admin",
		Token:             "",
		SkipSslValidation: false,
		HttpClient:        http.DefaultClient,
		UserAgent:         "Go-CF-client/1.1",
	}
}

func DefaultEndpoint() *Endpoint {
	return &Endpoint{
		DopplerEndpoint: "wss://doppler.10.244.0.34.xip.io:443",
		LoggingEndpoint: "wss://loggregator.10.244.0.34.xip.io:443",
		TokenEndpoint:   "https://uaa.10.244.0.34.xip.io",
		AuthEndpoint:    "https://login.10.244.0.34.xip.io",
	}
}

// NewClient returns a new client
func NewClient(config *Config) (client *Client, err error) {
	// bootstrap the config
	defConfig := DefaultConfig()

	if len(config.ApiAddress) == 0 {
		config.ApiAddress = defConfig.ApiAddress
	}

	if len(config.Username) == 0 {
		config.Username = defConfig.Username
	}

	if len(config.Password) == 0 {
		config.Password = defConfig.Password
	}

	if len(config.Token) == 0 {
		config.Token = defConfig.Token
	}

	if len(config.UserAgent) == 0 {
		config.UserAgent = defConfig.UserAgent
	}

	if config.HttpClient == nil {
		config.HttpClient = defConfig.HttpClient
	}

	if config.HttpClient.Transport == nil {
		config.HttpClient.Transport = shallowDefaultTransport()
	}

	tp := config.HttpClient.Transport.(*http.Transport)
	if tp.TLSClientConfig == nil {
		tp.TLSClientConfig = &tls.Config{}
	}

	// we want to keep the Timeout value from config.HttpClient
	timeout := config.HttpClient.Timeout

	ctx := context.Background()

	tp.TLSClientConfig.InsecureSkipVerify = config.SkipSslValidation
	ctx = context.WithValue(ctx, oauth2.HTTPClient, config.HttpClient)

	endpoint, err := getInfo(config.ApiAddress, oauth2.NewClient(ctx, nil))

	if err != nil {
		return nil, errors.Wrap(err, "Could not get api /v2/info")
	}

	switch {
	case config.Token != "":
		config = getUserTokenAuth(config, endpoint, ctx)
	case config.ClientID != "":
		config = getClientAuth(config, endpoint, ctx)
	default:
		config, err = getUserAuth(config, endpoint, ctx)
		if err != nil {
			return nil, err
		}
	}
	// make sure original Timeout value will be used
	if config.HttpClient.Timeout != timeout {
		config.HttpClient.Timeout = timeout
	}
	client = &Client{
		Config:   *config,
		Endpoint: *endpoint,
	}
	return client, nil
}

func shallowDefaultTransport() *http.Transport {
	defaultTransport := http.DefaultTransport.(*http.Transport)
	return &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
	}
}

func getUserAuth(config *Config, endpoint *Endpoint, ctx context.Context) (*Config, error) {
	authConfig := &oauth2.Config{
		ClientID: "cf",
		Scopes:   []string{""},
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoint.AuthEndpoint + "/oauth/auth",
			TokenURL: endpoint.TokenEndpoint + "/oauth/token",
		},
	}

	token, err := authConfig.PasswordCredentialsToken(ctx, config.Username, config.Password)

	if err != nil {
		return nil, errors.Wrap(err, "Error getting token")
	}

	config.TokenSource = authConfig.TokenSource(ctx, token)
	config.HttpClient = oauth2.NewClient(ctx, config.TokenSource)

	return config, err
}

func getClientAuth(config *Config, endpoint *Endpoint, ctx context.Context) *Config {
	authConfig := &clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     endpoint.TokenEndpoint + "/oauth/token",
	}

	config.TokenSource = authConfig.TokenSource(ctx)
	config.HttpClient = authConfig.Client(ctx)
	return config
}

// Initialize client credentials from existing bearer token
func getUserTokenAuth(config *Config, endpoint *Endpoint, ctx context.Context) *Config {
	authConfig := &oauth2.Config{
		ClientID: "cf",
		Scopes:   []string{""},
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoint.AuthEndpoint + "/oauth/auth",
			TokenURL: endpoint.TokenEndpoint + "/oauth/token",
		},
	}

	// Token is expected to have no "bearer" prefix
	token := &oauth2.Token{
		AccessToken: config.Token,
		TokenType:   "Bearer"}

	config.TokenSource = authConfig.TokenSource(ctx, token)
	config.HttpClient = oauth2.NewClient(ctx, config.TokenSource)

	return config
}

func getInfo(api string, httpClient *http.Client) (*Endpoint, error) {
	var endpoint Endpoint

	if api == "" {
		return DefaultEndpoint(), nil
	}

	resp, err := httpClient.Get(api + "/v2/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = decodeBody(resp, &endpoint)
	if err != nil {
		return nil, err
	}

	return &endpoint, err
}

// NewRequest is used to create a new request
func (c *Client) NewRequest(method, path string) *request {
	r := &request{
		method: method,
		url:    c.Config.ApiAddress + path,
		params: make(map[string][]string),
	}
	return r
}

// NewRequestWithBody is used to create a new request with
// arbigtrary body io.Reader.
func (c *Client) NewRequestWithBody(method, path string, body io.Reader) *request {
	r := c.NewRequest(method, path)

	// Set request body
	r.body = body

	return r
}

// DoRequest runs a request with our client
func (c *Client) DoRequest(r *request) (*http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.Config.UserAgent)
	if r.body != nil {
		req.Header.Set("Content-type", "application/json")
	}

	resp, err := c.Config.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var cfErr CloudFoundryError
		if err := decodeBody(resp, &cfErr); err != nil {
			return resp, errors.Wrap(err, "Unable to decode body")
		}
		return nil, cfErr
	}

	return resp, nil
}

// toHTTP converts the request to an HTTP request
func (r *request) toHTTP() (*http.Request, error) {

	// Check if we should encode the body
	if r.body == nil && r.obj != nil {
		b, err := encodeBody(r.obj)
		if err != nil {
			return nil, err
		}
		r.body = b
	}

	// Create the HTTP request
	return http.NewRequest(r.method, r.url, r.body)
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *Client) GetToken() (string, error) {
	token, err := c.Config.TokenSource.Token()
	if err != nil {
		return "", errors.Wrap(err, "Error getting bearer token")
	}
	return "bearer " + token.AccessToken, nil
}
