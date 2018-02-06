// Copyright © 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
)

var (
	expectedErr = errors.New("expected error")
)

// TestCreateBrokerFailure tests that errors in CreateBroker are returned correctly.
func TestCreateBrokerFailure(t *testing.T) {
	testFailure(t, func(adapter HttpAdapter) (int, []byte, error) {
		return adapter.CreateBroker(&CreateBrokerParams{})
	})
}

// TestCreateBrokerSuccess tests that the request body contains the broker
// and that the response is unmarshalled correctly.
func TestCreateBrokerSuccess(t *testing.T) {
	params := &CreateBrokerParams{
		Name:     "broker",
		Title:    "Success broker",
		Project:  "success",
		Catalogs: []string{"kit", "the", "kat"},
	}
	url := "https://www.happifying.com"
	resultBroker := Broker{
		Catalogs: params.Catalogs,
		Name:     fmt.Sprintf("projects/%s/brokers/%s", params.Project, params.Name),
		Title:    params.Title,
		URL:      &url,
	}

	client := &MockDoClient{
		do: func(req *http.Request) (*http.Response, error) {
			// Check request body.
			if req.Body == nil {
				t.Fatal("Request body was nil, but it should have the broker")
			}
			reqBody, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("Error reading request body: %v", err)
			}

			// Convert to broker.
			broker := Broker{}
			err = json.Unmarshal(reqBody, &broker)
			if err != nil {
				t.Fatalf("Error unmarshalling request body %s into broker: %v", string(reqBody), err)
			}

			// Check if broker matches. Note that there should be no URL.
			expectedBroker := resultBroker
			expectedBroker.URL = nil
			if !reflect.DeepEqual(expectedBroker, broker) {
				t.Fatalf("Brokers do not match: got %v; want %v", broker, expectedBroker)
			}

			resBody, err := json.Marshal(resultBroker)
			if err != nil {
				t.Fatalf("Error marshalling response body. This should never happen: %v", err)
			}

			return &http.Response{
				Body:       ioutil.NopCloser(bytes.NewReader(resBody)),
				StatusCode: 200,
			}, nil
		},
	}

	adapter := NewHttpAdapter(client)
	_, res, err := adapter.CreateBroker(params)
	if err != nil {
		t.Fatalf("Unexpected error from CreateBroker: %v", err)
	}

	broker := &Broker{}
	err = json.Unmarshal(res, broker)
	if err != nil {
		t.Fatalf("Error unmarshalling response body %s into broker: %v", string(res), err)
	}

	if !reflect.DeepEqual(&resultBroker, broker) {
		t.Fatalf("Brokers did not match: got %v; want %v", broker, &resultBroker)
	}
}

// TestDoRequestSuccess tests the success case of doRequest, namely that it will
// correctly do the request and return the body.
func TestDoRequestSuccess(t *testing.T) {
	expectedReqBody := []byte("Can I please have some water?")
	expectedResBody := []byte("Lake")
	expectedMethod := http.MethodGet
	expectedURL := "https://www.google.com"

	client := &MockDoClient{
		do: func(req *http.Request) (*http.Response, error) {
			// Check that URL matches.
			if gotURL := req.URL.String(); gotURL != expectedURL {
				t.Errorf("URLs did not match. Got %s but want %s", gotURL, expectedURL)
			}
			// Check that method matches.
			if req.Method != expectedMethod {
				t.Errorf("Request methods did not match. Got %s but want %s", req.Method, expectedMethod)
			}
			// Check that request body matches.
			if req.Body == nil {
				t.Errorf("Request body empty but want: %v", string(expectedReqBody))
			}
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("Unexpected error when reading request body: %v", err)
			}
			if !bytes.Equal(body, expectedReqBody) {
				t.Errorf("Request body doesn't match. Got %v but want %v", body, expectedReqBody)
			}

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewReader(expectedResBody)),
			}, nil
		},
	}

	adapter := NewHttpAdapter(client)
	_, res, err := adapter.doRequest(expectedURL, expectedMethod, bytes.NewReader(expectedReqBody))
	if err != nil {
		t.Fatalf("Unexpected error from doRequest: %v", err)
	}
	if !bytes.Equal(res, expectedResBody) {
		t.Errorf("Response body did not match. Got %s but want %s", string(res), string(expectedResBody))
	}
}

// TestDoRequestFailure tests that errors are correctly being returned in doRequest.
func TestDoRequestFailure(t *testing.T) {
	type testCase struct {
		// Parameters to pass or return to doRequest.
		method       string
		responseBody io.ReadCloser
		statusCode   int
		err          error
	}

	expectedErr := expectedErr

	do := func(t *testCase) (int, []byte, error) {
		adapter := NewHttpAdapter(&MockDoClient{
			do: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       t.responseBody,
					StatusCode: t.statusCode,
				}, t.err
			},
		})

		return adapter.doRequest("", t.method, nil)
	}

	// Here is a valid test case to make sure that tests are only failing
	// due to the reason we expect.
	validTestCase := testCase{
		method:       http.MethodGet,
		responseBody: ioutil.NopCloser(bytes.NewReader([]byte{})),
		statusCode:   200,
	}

	_, _, err := do(&validTestCase)
	if err != nil {
		t.Fatalf("Unexpected error in valid test case: %v", err)
	}

	// Check invalid method.
	badTestCase := validTestCase
	badTestCase.method = "functions over methods"
	if _, _, err = do(&badTestCase); err == nil {
		t.Error("Got no error but want one when using bad method")
	}

	// Check that request failures cause errors.
	badTestCase = validTestCase
	badTestCase.err = expectedErr
	if _, _, err = do(&badTestCase); err == nil {
		t.Error("Got no error but want one when returning error in request")
	}
	// Check that invalid response body causes error.
	badTestCase = validTestCase
	pipeReader, _ := io.Pipe()
	pipeReader.CloseWithError(expectedErr)
	badTestCase.responseBody = pipeReader
	if _, _, err = do(&badTestCase); err == nil {
		t.Error("Got no error but want one when using invalid response body")
	}

	// check that bad status codes cause errors.
	badTestCase = validTestCase
	badTestCase.statusCode = 500
	if _, _, err = do(&badTestCase); err == nil {
		t.Error("Got no error but want one when using bad status code")
	}
}

type MockDoClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (d *MockDoClient) Do(req *http.Request) (*http.Response, error) {
	return d.do(req)
}

func testFailure(t *testing.T, f func(HttpAdapter) (int, []byte, error)) {
	adapter := NewHttpAdapter(&MockDoClient{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, expectedErr
		},
	})
	_, _, err := f(*adapter)
	if err == nil {
		t.Fatal("Expected error but none found")
	}
}

func testSuccess(t *testing.T, expectedMethod, expectedURL string, f func(HttpAdapter) (int, []byte, error)) {
	expectedBody := []byte("I much prefer using the error monad to having to check a function's error return value")
	client := &MockDoClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.Method != expectedMethod {
				t.Fatalf("Request method got %s, want %s", req.Method, http.MethodDelete)
			} else if req.URL.String() != expectedURL {
				t.Fatalf("Request URL got %s, want %s", req.URL.String(), expectedURL)
			}
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewReader(expectedBody)),
			}, nil
		},
	}

	adapter := NewHttpAdapter(client)
	_, res, err := f(*adapter)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(res, expectedBody) {
		resString := "nil"
		if res != nil {
			resString = string(res)
		}
		t.Fatalf("Response got %v, want %v", resString, string(expectedBody))
	}
}
