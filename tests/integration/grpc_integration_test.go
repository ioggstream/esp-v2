// Copyright 2018 Google Cloud Platform Proxy Authors
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

package integration

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
)

const (
	addr = "127.0.0.1:8080"
)

var successTrailer, abortedTrailer, dataLossTrailer, internalTrailer client.GRPCWebTrailer

func init() {
	successTrailer = client.GRPCWebTrailer{"grpc-message": "OK", "grpc-status": "0"}
	abortedTrailer = client.GRPCWebTrailer{"grpc-message": "ABORTED", "grpc-status": "10"}
	internalTrailer = client.GRPCWebTrailer{"grpc-message": "INTERNAL", "grpc-status": "13"}
	dataLossTrailer = client.GRPCWebTrailer{"grpc-message": "DATA_LOSS", "grpc-status": "15"}
}

func TestGRPC(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv( /*mockMetadata=*/ true /*mockServiceManagement=*/, true /*mockServiceControl=*/, true /*mockJwtPrividers=*/, nil)

	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc           string
		clientProtocol string
		method         string
		wantResp       string
	}{
		{
			desc:           "gRPC client calling gRPC backend",
			clientProtocol: "grpc",
			method:         "GetShelf",
			wantResp:       `{"theme":"Unknown Shelf"}`,
		},
		{
			desc:           "Http client calling gRPC backend",
			clientProtocol: "http",
			method:         "/v1/shelves/125",
			wantResp:       `{"id":"125","theme":"Unknown Shelf"}`,
		},
	}

	for _, tc := range tests {
		resp, err := client.MakeCall(tc.clientProtocol, addr, "GET", tc.method, "")
		if err != nil {
			t.Errorf("failed to run test: %s", err)
		}

		if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}
	}
}

func TestGRPCWeb(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=grpc", "--rollout_strategy=fixed"}
	s := env.NewTestEnv( /*mockMetadata=*/ true /*mockServiceManagement=*/, true /*mockServiceControl=*/, true /*mockJwtPrividers=*/, nil)

	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc          string
		method        string
		grpcTestValue string
		wantResp      string
		wantTrailer   client.GRPCWebTrailer
	}{
		// Successes:
		{
			method:      "ListShelves",
			wantResp:    `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
			wantTrailer: successTrailer,
		},
		{
			method:      "DeleteShelf",
			wantResp:    "{}",
			wantTrailer: successTrailer,
		},
		{
			method:      "GetShelf",
			wantResp:    `{"theme":"Unknown Shelf"}`,
			wantTrailer: successTrailer,
		},
		// Failures:
		{
			method:        "GetShelf",
			grpcTestValue: "ABORTED",
			wantTrailer:   abortedTrailer,
		},
		{
			method:        "DeleteShelf",
			grpcTestValue: "INTERNAL",
			wantTrailer:   internalTrailer,
		},
		{
			method:        "ListShelves",
			grpcTestValue: "DATA_LOSS",
			wantTrailer:   dataLossTrailer,
		},
	}

	for _, tc := range tests {
		grpcTestValues := []string{}
		if tc.grpcTestValue != "" {
			grpcTestValues = []string{tc.grpcTestValue}
		}

		resp, trailer, err := client.MakeGRPCWebCall(addr, tc.method, "", grpcTestValues...)

		if err != nil {
			t.Errorf("failed to run test: %s", err)
		}

		if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.method, tc.wantResp, resp)
		}

		if !reflect.DeepEqual(trailer, tc.wantTrailer) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.method, tc.wantTrailer, trailer)

		}
	}
}

func TestGRPCJwt(t *testing.T) {
	serviceName := "bookstore-service"
	configID := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--skip_service_control_filter=true", "--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv( /*mockMetadata=*/ true /*mockServiceManagement=*/, true /*mockServiceControl=*/, true /*mockJwtPrividers=*/, []string{"google_service_account", "endpoints_jwt", "broken_provider"})
	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		token              string
		wantResp           string
		wantError          string
		wantGRPCWebError   string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}{
		// Testing JWT is required or not.
		{
			desc:             "Fail for gPRC client, without valid JWT token",
			clientProtocol:   "grpc",
			method:           "ListShelves",
			wantError:        "code = Unauthenticated desc = Jwt is missing",
			wantGRPCWebError: "401 Unauthorized",
		},
		{
			desc:           "Fail for Http client, without valid JWT token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			wantError:      "401 Unauthorized",
		},
		{
			desc:           "Succeed for Http client, JWT rule recognizes {shelf} correctly",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/25",
			wantResp:       `{"id":"25","theme":"Unknown Shelf"}`,
		},
		{
			desc:             "Fail for gPRC client, with bad JWT token",
			clientProtocol:   "grpc",
			method:           "ListShelves",
			token:            testdata.FakeBadToken,
			wantError:        "code = Unauthenticated desc = Jwt issuer is not configured",
			wantGRPCWebError: "401 Unauthorized",
		},
		{
			desc:           "Fail for Http client, with bad JWT token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			token:          testdata.FakeBadToken,
			wantError:      "401 Unauthorized",
		},
		{
			desc:           "Succeed for Http client, with valid JWT token, with url binding",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves?shelf.id=123",
			token:          testdata.FakeCloudToken,
			wantResp:       `{"id":"123","theme":"New Shelf"}`,
		},
		{
			desc:               "Succeed for gPRC client, with valid JWT token",
			clientProtocol:     "grpc",
			method:             "CreateShelf",
			token:              testdata.FakeCloudToken,
			wantResp:           `{"theme":"New Shelf"}`,
			wantGRPCWebTrailer: successTrailer,
		},

		// Testing JWT RouteMatcher matches by HttpHeader and parameters in "{}", for Http Client only.
		{
			desc:           "Succeed for Http client, Jwt RouteMatcher matches by HttpHeader method",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves?shelf.id=345&shelf.theme=HurryUp",
			token:          testdata.FakeCloudToken,
			wantResp:       `{"id":"345","theme":"HurryUp"}`,
		},
		{
			desc:           "Fail for Http client, Jwt RouteMatcher matches by HttpHeader method",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves",
			wantError:      "401 Unauthorized",
		},
		{
			desc:           "Succeed for Http client, Jwt RouteMatcher works for multi query parameters",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/125/books/001",
			token:          testdata.FakeCloudToken,
			wantResp:       "{}",
		},
		{
			desc:           "Fail for Http client, Jwt RouteMatcher works for multi query parameters",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/125/books/001",
			wantError:      "401 Unauthorized",
		},
		{
			desc:           "Succeed for Http client, Jwt RouteMatcher works for multi query parameters and HttpHeader, no audience",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/125/books/12345",
			wantResp:       `{"id":"12345","title":"Unknown Book"}`,
		},

		// Test JWT with audiences.
		{
			desc:               "Succeed for gPRC client, with valid JWT token, with single audience",
			clientProtocol:     "grpc",
			method:             "ListShelves",
			token:              testdata.FakeCloudTokenSingleAudience1,
			wantResp:           `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
			wantGRPCWebTrailer: successTrailer,
		},
		{
			desc:           "Succeed for Http client, with valid JWT token, with single audience",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			token:          testdata.FakeCloudTokenSingleAudience1,
			wantResp:       `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
		},
		{
			desc:             "Fail for gPRC client, with JWT token but not expected audience",
			clientProtocol:   "grpc",
			method:           "ListShelves",
			token:            testdata.FakeCloudToken,
			wantError:        "code = Unauthenticated desc = Audiences in Jwt are not allowed",
			wantGRPCWebError: "401 Unauthorized",
		},
		{
			desc:           "Fail for Http client, with JWT token but not expected audience",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			token:          testdata.FakeCloudToken,
			wantError:      "401 Unauthorized",
		},
		{
			desc:             "Fail for gPRC client, with JWT token but wrong audience",
			clientProtocol:   "grpc",
			method:           "ListShelves",
			token:            testdata.FakeCloudTokenSingleAudience2,
			wantError:        "code = Unauthenticated desc = Audiences in Jwt are not allowed",
			wantGRPCWebError: "401 Unauthorized",
		},
		{
			desc:               "Succeed for gPRC client, with JWT token with one audience while multi audiences are allowed",
			clientProtocol:     "grpc",
			method:             "CreateBook",
			token:              testdata.FakeCloudTokenSingleAudience2,
			wantResp:           `{"title":"New Book"}`,
			wantGRPCWebTrailer: successTrailer,
		},
		{
			desc:           "Succeed for Http client, with JWT token with multi audience while multi audiences are allowed",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/12345/books",
			token:          testdata.FakeCloudTokenMultiAudiences,
			wantResp:       `{"id":"12345","title":"New Book"}`,
		},

		// Testing JWT with multiple Providers, token from anyone should work,
		// even with an invalid issuer.
		{
			desc:           "Succeed for Http client, with multi requirements from different providers",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/120",
			token:          testdata.FakeEndpointsToken,
			wantResp:       "{}",
		},
		{
			desc:               "Succeed for gPRC client, with multi requirements from different providers",
			clientProtocol:     "grpc",
			method:             "DeleteShelf",
			token:              testdata.FakeCloudTokenSingleAudience1,
			wantResp:           "{}",
			wantGRPCWebTrailer: successTrailer,
		},
		{
			desc:           "Fail for Http client, with multi requirements from different providers",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/120",
			token:          testdata.FakeCloudToken,
			wantError:      "401 Unauthorized",
		},
	}

	for _, tc := range tests {
		resp, err := client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token)

		if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}

		// For grpc, also test gRPC-web variant.
		if tc.clientProtocol != "grpc" {
			continue
		}

		grpcWebDesc := strings.Replace(tc.desc, "gRPC", "gRPC-Web", -1)
		grpcWebResp, trailer, err := client.MakeGRPCWebCall(addr, tc.method, tc.token)
		if tc.wantGRPCWebError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantGRPCWebError)) {
			t.Errorf("Test (%s): failed\n  expected: %v\n  got: %v", grpcWebDesc, tc.wantGRPCWebError, err)
		}

		if tc.wantResp != "" && !strings.Contains(grpcWebResp, tc.wantResp) {
			t.Errorf("Test (%s): failed\n  expected: %s\n  got: %s", grpcWebDesc, tc.wantResp, grpcWebResp)
		}

		if !reflect.DeepEqual(trailer, tc.wantGRPCWebTrailer) {
			t.Errorf("Test (%s): failed\n  expected: %s\n  got: %s", grpcWebDesc, tc.wantGRPCWebTrailer, trailer)
		}
	}
}