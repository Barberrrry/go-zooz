package zooz

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/pkg/errors"
)

type httpClientMock struct {
	do func(r *http.Request) (*http.Response, error)
}

type request struct {
	Field string `json:"field"`
}

func (c *httpClientMock) Do(r *http.Request) (*http.Response, error) {
	return c.do(r)
}

func TestCall_WithApiResponse(t *testing.T) {
	httpClientMock := &httpClientMock{
		do: func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.paymentsos.com/somepath" {
				t.Errorf("Invalid request URI: %s", r.RequestURI)
			}
			if r.Method != "POST" {
				t.Errorf("Invalid request method: %s", r.Method)
			}
			if r.Header.Get(headerEnv) != string(EnvTest) {
				t.Errorf("Invalid request env: %s", r.Header.Get(headerEnv))
			}
			if r.Header.Get(headerAppID) != "app_id_test" {
				t.Errorf("Invalid request app ID: %s", r.Header.Get(headerAppID))
			}
			if r.Header.Get(headerPrivateKey) != "private_key_test" {
				t.Errorf("Invalid request private key: %s", r.Header.Get(headerPrivateKey))
			}
			if r.Header.Get("test-header") != "test-header-value" {
				t.Errorf("Invalid request custom header: %s", r.Header.Get("test-header"))
			}
			body, _ := ioutil.ReadAll(r.Body)
			if string(body) != `{"field":"request_value"}` {
				t.Errorf("Invalid request body: %s", string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"field":"response_value"}`)),
			}, nil
		},
	}

	request := request{
		Field: "request_value",
	}

	response := struct {
		Field string `json:"field"`
	}{}

	client := Client{
		httpClient: httpClientMock,
		appID:      "app_id_test",
		privateKey: "private_key_test",
		env:        EnvTest,
	}

	err := client.Call(
		context.Background(),
		"POST",
		"somepath",
		map[string]string{
			"test-header": "test-header-value",
		},
		&request,
		&response,
	)

	if err != nil {
		t.Errorf("Call returned error: %v", err)
	}

	if response.Field != "response_value" {
		t.Errorf("Response is invalid: %+v", response)
	}
}

func TestCall_WithApiError(t *testing.T) {
	httpClientMock := &httpClientMock{
		do: func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"category":"category_test"}`)),
			}, nil
		},
	}

	request := request{
		Field: "request_value",
	}

	client := Client{httpClient: httpClientMock}

	err := client.Call(
		context.Background(),
		"POST",
		"somepath",
		map[string]string{
			"test-header": "test-header-value",
		},
		&request,
		nil,
	)

	if err == nil {
		t.Error("Call didn't return error")
	}
	if zoozErr, ok := err.(*Error); ok {
		if zoozErr.StatusCode != http.StatusBadRequest {
			t.Errorf("Invalid error status code: %d", zoozErr.StatusCode)
		}
		if zoozErr.ApiError.Category != "category_test" {
			t.Errorf("Invalid API error category: %d", zoozErr.ApiError.Category)
		}
	} else {
		t.Errorf("Call return invalid error type: %T", err)
	}
}

func TestCall_WithTransportError(t *testing.T) {
	httpClientMock := &httpClientMock{
		do: func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("do_error")
		},
	}

	request := request{
		Field: "request_value",
	}

	client := Client{httpClient: httpClientMock}

	err := client.Call(
		context.Background(),
		"POST",
		"somepath",
		map[string]string{
			"test-header": "test-header-value",
		},
		&request,
		nil,
	)

	if err == nil {
		t.Error("Call didn't return error")
	}
	if errors.Cause(err).Error() != "do_error" {
		t.Errorf("Invalid error cause: %v", errors.Cause(err))
	}
}
