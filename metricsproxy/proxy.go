package metricsproxy

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

import (
	"code.cloudfoundry.org/lager"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// credit: https://eli.thegreenplace.net/2022/go-and-proxy-servers-part-1-http-proxies/ (https://archive.is/dpI4I)

//counterfeiter:generate . MetricsProxy
type MetricsProxy interface {
	ForwardRequestToURL(w http.ResponseWriter, req *http.Request,
		appURL *url.URL, headers map[string]string)
}

type metricsProxy struct {
	httpclient *http.Client
	logger     lager.Logger
}

type Config struct {
	Logger     lager.Logger
	HttpClient *http.Client
}

func New(config Config) MetricsProxy {
	if config.HttpClient == nil {
		config.HttpClient = &http.Client{}
	}
	proxy := metricsProxy{
		httpclient: config.HttpClient,
		logger:     config.Logger,
	}
	return &proxy
}

func (p *metricsProxy) ForwardRequestToURL(w http.ResponseWriter, req *http.Request,
	appURL *url.URL, headers map[string]string) {
	// When a http.Request is sent through an http.Client, RequestURI should not
	// be set (see documentation of this field).
	req.RequestURI = ""

	removeHopHeaders(req.Header)
	removeConnectionHeaders(req.Header)

	req.URL = appURL
	req.Host = appURL.Host

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	p.logger.Debug("addheaders", lager.Data{"headers": req.Header})

	resp, err := p.httpclient.Do(req)
	if err != nil {
		p.logger.Error("doremoterequest", err, lager.Data{"url": appURL, "headers": headers})
		http.Error(w, "error making request", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	removeHopHeaders(resp.Header)
	removeConnectionHeaders(resp.Header)

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return
	}
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Error copying response body", http.StatusInternalServerError)
	}
}

var hopHeaders = []string{
	"Connection",
	"metricsProxy-Connection",
	"Keep-Alive",
	"metricsProxy-Authenticate",
	"metricsProxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // spelling per https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func removeHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection"
// header of h. See RFC 7230, section 6.1
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}
