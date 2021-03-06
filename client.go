// Package zooz contains Go client for Zooz API.
//
// Zooz API documentation: https://developers.paymentsos.com/docs/api
//
// Before using this client you need to register and configure Zooz account: https://developers.paymentsos.com/docs/quick-start.html
package zooz

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// Call makes HTTP call with given options and decode response into given struct.
// Client implements this interface and pass itself to entity clients. You may create entity clients with own caller for
// test purposes.
type Caller interface {
	Call(ctx context.Context, method, path string, headers map[string]string, reqObj interface{}, respObj interface{}) error
}

// httpClient is internal interface fot HTTP client. Built-in net/http.Client implements this interface as well.
type httpClient interface {
	Do(r *http.Request) (*http.Response, error)
}

// Option is a callback for redefine client parameters.
type Option func(*Client)

// Client contains API parameters and provides set of API entity clients.
type Client struct {
	httpClient httpClient
	appID      string
	privateKey string
	env        env
}

type env string

const (
	apiVersion = "1.0.0"
	apiUrl     = "https://api.paymentsos.com/"

	// Possible values for environment request header
	EnvTest env = "test"
	EnvLive env = "live"

	headerApiVersion      = "api-version"
	headerEnv             = "x-payments-os-env"
	headerIdempotencyKey  = "idempotency_key"
	headerAppID           = "app_id"
	headerPrivateKey      = "private_key"
	headerClientIpAddress = "x-client-ip-address"
	headerClientUserAgent = "x-client-user-agent"
	headerRequestID       = "X-Zooz-Request-Id"
)

// New creates new client with given options.
func New(options ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		env:        EnvTest,
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// OptHttpClient returns option with given HTTP client.
func OptHttpClient(httpClient httpClient) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// OptAppID returns option with given App ID.
func OptAppID(appID string) Option {
	return func(c *Client) {
		c.appID = appID
	}
}

// OptPrivateKey returns option with given private key.
func OptPrivateKey(privateKey string) Option {
	return func(c *Client) {
		c.privateKey = privateKey
	}
}

// OptEnv returns option with given environment value.
func OptEnv(env env) Option {
	return func(c *Client) {
		c.env = env
	}
}

// Call does HTTP request with given params using set HTTP client. Response will be decoded into respObj.
// Error may be returned if something went wrong. If API return error as response, then Call returns error of type zooz.Error.
func (c *Client) Call(ctx context.Context, method, path string, headers map[string]string, reqObj interface{}, respObj interface{}) error {
	var reqBody []byte
	var err error

	if reqObj != nil {
		reqBody, err = json.Marshal(reqObj)
		if err != nil {
			return errors.Wrap(err, "failed to marshal request body")
		}
	}

	req, err := http.NewRequest(method, apiUrl+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	req = req.WithContext(ctx)

	// Set call-specific headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set common client headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerApiVersion, apiVersion)
	req.Header.Set(headerEnv, string(c.env))
	req.Header.Set(headerAppID, c.appID)
	req.Header.Set(headerPrivateKey, c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to do request")
	}
	defer resp.Body.Close()

	// Handle 4xx and 5xx statuses
	if resp.StatusCode >= http.StatusBadRequest {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}
		var apiError ApiError
		if err := json.Unmarshal(respBody, &apiError); err != nil {
			return errors.Wrapf(err, "failed to unmarshal response error with status %d: %s", resp.StatusCode, string(respBody))
		}
		return &Error{
			StatusCode: resp.StatusCode,
			RequestID:  resp.Header.Get(headerRequestID),
			ApiError:   apiError,
		}
	}

	// Decode response into a struct if it was given
	if respObj != nil {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}
		if err := json.Unmarshal(respBody, respObj); err != nil {
			return errors.Wrapf(err, "failed to unmarshal response body: %s", string(respBody))
		}
	}

	return nil
}

// Payment creates client for work with corresponding entity.
func (c *Client) Payment() *PaymentClient {
	return &PaymentClient{Caller: c}
}

// Customer creates client for work with corresponding entity.
func (c *Client) Customer() *CustomerClient {
	return &CustomerClient{Caller: c}
}

// PaymentMethod creates client for work with corresponding entity.
func (c *Client) PaymentMethod() *PaymentMethodClient {
	return &PaymentMethodClient{Caller: c}
}

// Authorization creates client for work with corresponding entity.
func (c *Client) Authorization() *AuthorizationClient {
	return &AuthorizationClient{Caller: c}
}

// Charge creates client for work with corresponding entity.
func (c *Client) Charge() *ChargeClient {
	return &ChargeClient{Caller: c}
}

// Capture creates client for work with corresponding entity.
func (c *Client) Capture() *CaptureClient {
	return &CaptureClient{Caller: c}
}

// Void creates client for work with corresponding entity.
func (c *Client) Void() *VoidClient {
	return &VoidClient{Caller: c}
}

// Refund creates client for work with corresponding entity.
func (c *Client) Refund() *RefundClient {
	return &RefundClient{Caller: c}
}

// Redirection creates client for work with corresponding entity.
func (c *Client) Redirection() *RedirectionClient {
	return &RedirectionClient{Caller: c}
}
