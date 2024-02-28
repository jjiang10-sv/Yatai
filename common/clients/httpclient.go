package clients

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-querystring/query"

	contextutil "github.com/bentoml/yatai/common/utils/context"
	"github.com/bentoml/yatai/common/utils/metrics"
)

const (
	defaultRestTimeout           = 5 * time.Second
	defaultMaxConnectionsPerHost = 100
	defaultKeepAliveTimeout      = 60 * time.Second
	defaultIdleConnTimeout       = 30 * time.Second
	defaultMaxConnsPerHost       = 100

	CtxHttpMethod       contextKey = "CTX_HTTP_METHOD"
	CtxHttpHost         contextKey = "CTX_HTTP_HOST"
	CtxHttpPathTemplate contextKey = "CTX_HTTP_PATH_TEMPLATE"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HttpClient struct {
	Host       string
	HttpClient httpClient
}

type Config struct {
	Host             string
	Cert             string
	Key              string
	MaxConnsPerHost  *int
	KeepAliveTimeout *time.Duration
	RestTimeout      *time.Duration
	IdleConnTimeout  *time.Duration
}

type contextKey string

type RequestError struct {
	StatusCode int
	Err        error
}

// TODO: abstract out into generic helper package
func (r *RequestError) Error() string {
	return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Err.Error())
}

func (r *RequestError) Temporary() bool {
	return r.StatusCode == http.StatusServiceUnavailable // 503
}

func (r *RequestError) IsOK() bool {
	return r.StatusCode == http.StatusOK // 200
}

func NewHttpClient(config Config) (*HttpClient, error) {
	httpClient, err := setupHTTPClient(config)
	if err != nil {
		return nil, err
	}

	return &HttpClient{
		Host:       config.Host,
		HttpClient: httpClient,
	}, nil
}

// NewHttpClientSimple create HttpClient with default value
func NewHttpClientSimple() (*HttpClient, error) {
	httpClient, err := setupHTTPClient(Config{})
	if err != nil {
		return nil, err
	}

	return &HttpClient{
		HttpClient: httpClient,
	}, nil
}

// initialize an HTTP client for a xxx internal service with config NewInternalClientConfig
func NewInternalClientConfig(host, cert, key string) (*Config, error) {
	if host == "" {
		return nil, fmt.Errorf("Host not provided for Internal HTTP Client")
	}
	if cert == "" {
		return nil, fmt.Errorf("Cert not provided for Internal HTTP Client")
	}
	if key == "" {
		return nil, fmt.Errorf("Key not provided for Internal HTTP Client")
	}
	return &Config{
		Host: host,
		Cert: cert,
		Key:  key,
	}, nil
}

// initialize an HTTP client for an external service with config from NewExternalClientConfig
func NewExternalClientConfig(host string) (*Config, error) {
	if host == "" {
		return nil, fmt.Errorf("Host not provided for External HTTP Client")
	}
	return &Config{
		Host: host,
	}, nil
}

func setupHTTPClient(config Config) (*http.Client, error) {
	restTimeout := defaultRestTimeout
	keepAliveTimeout := defaultKeepAliveTimeout
	idleConnTimeout := defaultIdleConnTimeout
	maxConnsPerHost := defaultMaxConnsPerHost

	if config.RestTimeout != nil {
		restTimeout = *config.RestTimeout
	}
	if config.KeepAliveTimeout != nil {
		keepAliveTimeout = *config.KeepAliveTimeout
	}
	if config.IdleConnTimeout != nil {
		idleConnTimeout = *config.IdleConnTimeout
	}
	if config.MaxConnsPerHost != nil {
		maxConnsPerHost = *config.MaxConnsPerHost
	}

	cert, err := GetCert(config.Cert, config.Key)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: instrumentPromRoundTripper(&http.Transport{
			MaxConnsPerHost:     maxConnsPerHost,
			MaxIdleConnsPerHost: maxConnsPerHost,
			TLSHandshakeTimeout: restTimeout,
			Dial: (&net.Dialer{
				Timeout:   restTimeout,
				KeepAlive: keepAliveTimeout,
			}).Dial,
			IdleConnTimeout: idleConnTimeout,
			TLSClientConfig: generateTlsConfig(cert),
		}),
		Timeout: restTimeout,
	}, nil
}

// The below RoundTripper implementation was inspired from prometheus:
// https://github.com/prometheus/client_golang/blob/dc1559e8efad6493bfa27e5af5e9a030e8675d73/prometheus/promhttp/instrument_client.go#L28-L46

// The RoundTripperFunc type is an adapter to allow the use of ordinary
// functions as RoundTrippers. If f is a function with the appropriate
// signature, RountTripperFunc(f) is a RoundTripper that calls f.
type RoundTripperFunc func(req *http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface.
func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

func instrumentPromRoundTripper(next http.RoundTripper) RoundTripperFunc {
	return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		httpMethod := r.Context().Value(CtxHttpMethod)
		httpHost := r.Context().Value(CtxHttpHost)
		httpPathTemplate := r.Context().Value(CtxHttpPathTemplate)

		if httpMethod != nil && httpHost != nil && httpPathTemplate != nil {
			metrics.HTTPClientRecorder.IncInFlight(httpHost.(string), httpPathTemplate.(string))
			defer metrics.HTTPClientRecorder.DecInFlight(httpHost.(string), httpPathTemplate.(string))
			start := time.Now()

			resp, err := next.RoundTrip(r)
			if err == nil {
				metrics.HTTPClientRecorder.LogDuration(
					start,
					httpHost.(string),
					httpPathTemplate.(string),
					httpMethod.(string),
					resp.StatusCode,
				)
				return resp, err
			} else {
				return resp, err
			}
		} else {
			return next.RoundTrip(r)
		}
	})
}

func GetCert(certFile, keyFile string) (*tls.Certificate, error) {
	if certFile == "" || keyFile == "" {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func generateTlsConfig(cert *tls.Certificate) *tls.Config {
	if cert == nil {
		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}
}

// http Request prometheus helpers

// NewRequestWithProm creates a new http.Request while preserving the host and templated url for prometheus metrics
// using the background context
func NewRequestWithProm(
	method string,
	host string,
	pathTemplate string,
	pathMap map[string]interface{},
	query map[string]interface{},
	body io.Reader,
) (*http.Request, error) {
	return NewRequestWithPromContext(
		context.Background(),
		method,
		host,
		pathTemplate,
		pathMap,
		query,
		body,
	)
}

// NewRequestWithPromContext creates a new http.Request while preserving the host and templated url for prometheus
// metrics using the supplied context
func NewRequestWithPromContext(
	ctx context.Context,
	method string,
	host string,
	pathTemplate string,
	pathMap map[string]interface{},
	query map[string]interface{},
	body io.Reader,
) (*http.Request, error) {
	// String potentially trailing or leading slashes
	host = strings.TrimSuffix(host, "/")
	pathTemplate = strings.TrimPrefix(pathTemplate, "/")

	// Build the url string
	fqUrlString := fmt.Sprintf("%s/%s", host, pathTemplate)

	// Build the url object for use in metrics
	templateUrl, err := url.Parse(fqUrlString)
	if err != nil {
		return nil, fmt.Errorf("unable to construct template url for %s", fqUrlString)
	}

	// Substitute path placeholders with actual values
	for k, v := range pathMap {
		fqUrlString = strings.Replace(
			fqUrlString,
			fmt.Sprintf(":%s", k),
			fmt.Sprintf("%v", v),
			1,
		)
	}

	// Create the "real" url object to be used with request
	parsedUrl, err := url.Parse(fqUrlString)
	if err != nil {
		return nil, fmt.Errorf("unable to construct url for %s", fqUrlString)
	}

	// Create query params if they exist
	q := parsedUrl.Query()
	for k, v := range query {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	parsedUrl.RawQuery = q.Encode()

	// Set required context values for prometheus/RoundTripper
	ctx = context.WithValue(ctx, CtxHttpMethod, method)
	ctx = context.WithValue(ctx, CtxHttpHost, templateUrl.Host)
	ctx = context.WithValue(ctx, CtxHttpPathTemplate, templateUrl.Path)
	reqUrl := parsedUrl.String()
	if strings.Contains(reqUrl, "cloud.xxx.cn/sites?site_number=") {
		reqUrl = strings.Replace(reqUrl, "?site_number=", "/", 1)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqUrl, body)
	if err != nil {
		return nil, err
	}

	// set request id in context on all http requests
	requestID := contextutil.GetRequestIdFromContext(ctx)
	if requestID != "" {
		req.Header.Add("X-Request-ID", requestID)
	}
	return req, nil
}

// NewRequestWithPromContext creates a new http.Request while preserving the host and templated url for prometheus
// metrics using the supplied context
func NewRequestWithPromContextSpecial(
	ctx context.Context,
	method string,
	host string,
	pathTemplate string,
	pathMap map[string]interface{},
	query map[string]interface{},
	body io.Reader,
) (*http.Request, error) {

	ctxRes, reqUrl, err := setup(ctx, method, host, pathTemplate, pathMap, query)
	if err != nil {
		return nil, err
	}
	if strings.Contains(reqUrl, "cloud.xxx.cn/sites?site_number=") {
		reqUrl = strings.Replace(reqUrl, "?site_number=", "/", 1)
	}
	return postSetup(ctxRes, method, reqUrl, body)

}

func setup(ctx context.Context,
	method string,
	host string,
	pathTemplate string,
	pathMap map[string]interface{},
	query map[string]interface{},
) (ctxRes context.Context, reqUrl string, err error) {
	// String potentially trailing or leading slashes
	host = strings.TrimSuffix(host, "/")
	pathTemplate = strings.TrimPrefix(pathTemplate, "/")

	// Build the url string
	fqUrlString := fmt.Sprintf("%s/%s", host, pathTemplate)

	// Build the url object for use in metrics
	templateUrl, err := url.Parse(fqUrlString)
	if err != nil {
		return nil, "", fmt.Errorf("unable to construct template url for %s", fqUrlString)
	}

	// Substitute path placeholders with actual values
	for k, v := range pathMap {
		fqUrlString = strings.Replace(
			fqUrlString,
			fmt.Sprintf(":%s", k),
			fmt.Sprintf("%v", v),
			1,
		)
	}

	// Create the "real" url object to be used with request
	parsedUrl, err := url.Parse(fqUrlString)
	if err != nil {
		return nil, "", fmt.Errorf("unable to construct url for %s", fqUrlString)
	}

	// Create query params if they exist
	q := parsedUrl.Query()
	for k, v := range query {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	parsedUrl.RawQuery = q.Encode()

	// Set required context values for prometheus/RoundTripper

	ctxRes = context.WithValue(ctx, CtxHttpMethod, method)
	ctxRes = context.WithValue(ctxRes, CtxHttpHost, templateUrl.Host)
	ctxRes = context.WithValue(ctxRes, CtxHttpPathTemplate, templateUrl.Path)
	reqUrl = parsedUrl.String()
	return ctxRes, reqUrl, nil
}

func postSetup(ctxRes context.Context, method string, reqUrl string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctxRes, method, reqUrl, body)
	if err != nil {
		return nil, err
	}
	// set request id in context on all http requests
	requestID := contextutil.GetRequestIdFromContext(ctxRes)
	if requestID != "" {
		req.Header.Add("X-Request-ID", requestID)
	}
	return req, nil
}
func QueryFromInterface(i interface{}) (map[string]interface{}, error) {
	values, err := query.Values(i)
	if err != nil {
		return nil, fmt.Errorf("unable to convert %v into query params", i)
	}
	var urlQuery = make(map[string]interface{})
	for k, v := range values {
		//k = ""
		urlQuery[k] = strings.Join(v, "")
	}
	return urlQuery, nil
}
func InstrumentPromRoundTripper(next http.RoundTripper) RoundTripperFunc {
	return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		httpMethod := r.Context().Value(CtxHttpMethod)
		httpHost := r.Context().Value(CtxHttpHost)
		httpPathTemplate := r.Context().Value(CtxHttpPathTemplate)

		if httpMethod != nil && httpHost != nil && httpPathTemplate != nil {
			metrics.HTTPClientRecorder.IncInFlight(httpHost.(string), httpPathTemplate.(string))
			defer metrics.HTTPClientRecorder.DecInFlight(httpHost.(string), httpPathTemplate.(string))
			start := time.Now()

			resp, err := next.RoundTrip(r)
			if err == nil {
				metrics.HTTPClientRecorder.LogDuration(
					start,
					httpHost.(string),
					httpPathTemplate.(string),
					httpMethod.(string),
					resp.StatusCode,
				)
				return resp, err
			} else {
				return resp, err
			}
		} else {
			return next.RoundTrip(r)
		}
	})
}

func SetupHttpClient(ClientCert, ClientKey string) (newClient *metrics.HTTPClient, err error) {
	var tlsConfig = &tls.Config{}
	ServerCaCert := ""
	if ServerCaCert != "" {
		clientCACert, err := os.ReadFile(ServerCaCert)
		if err != nil {
			err = fmt.Errorf("Can't read cert file %v error is %v", ServerCaCert, err)
			return nil, err
		}
		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCACert)
		tlsConfig = &tls.Config{
			RootCAs: clientCertPool,
		}
	}

	if ClientCert != "" && ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(ClientCert, ClientKey)
		if err != nil {
			err = fmt.Errorf("can't LoadX509KeyPair, %v", err)
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		//tlsConfig.BuildNameToCertificate()
	}
	MaxConnectionsPerHost := defaultMaxConnectionsPerHost
	RestTimeout := defaultRestTimeout
	KeepAliveTimeout := defaultKeepAliveTimeout
	IdleConnTimeout := defaultIdleConnTimeout
	// logx.Info("Asset Max Connections Per Host: %v\n", MaxConnectionsPerHost)
	// logx.Info("Asset Timeout: %v\n", RestTimeout)
	client := &http.Client{
		Transport: InstrumentPromRoundTripper(&http.Transport{
			MaxConnsPerHost:     MaxConnectionsPerHost,
			MaxIdleConnsPerHost: MaxConnectionsPerHost,
			Dial: (&net.Dialer{
				KeepAlive: KeepAliveTimeout,
				Timeout:   RestTimeout,
			}).Dial,
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: RestTimeout,
			IdleConnTimeout:     IdleConnTimeout,
		}),
		Timeout: RestTimeout,
	}

	newClient = &metrics.HTTPClient{Client: client}

	return
}
