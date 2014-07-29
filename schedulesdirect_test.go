package schedulesdirect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// inspired by https://willnorris.com/2013/08/testing-in-go-github
var (
	mux    *http.ServeMux
	server *httptest.Server
	client sdclient
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	// schedules direct client configured to use test server
	client = sdclient{
		baseURL: server.URL,
	}
}

func testMethod(t *testing.T, r *http.Request, expectedMethod string) {
	if r.Method != expectedMethod {
		t.Fatalf("method (%s) != expectedMethod (%s)", r.Method, expectedMethod)
	}
}

func testHeader(t *testing.T, r *http.Request, header, expectedValue string) {
	if r.Header.Get(header) != expectedValue {
		t.Fatalf("token (%s) != expectedValue (%s)", r.Header.Get("token"), expectedValue)
	}
}

func TestGetTokenOK(t *testing.T) {
	setup()

	mux.HandleFunc("/token",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")

			var tokenReq tokenRequest

			errDecode := json.NewDecoder(r.Body).Decode(&tokenReq)
			if errDecode != nil {
				t.Fatal(errDecode)
			}

			fmt.Fprint(w, `{"code":0,"message":"OK","serverID":"serverID1","token":"token1"}`)
		},
	)

	token, errToken := client.GetToken("user1", "pass1")
	if errToken != nil {
		t.Fatal(errToken)
	}

	if token != "token1" {
		t.Fatalf("token doesn't match")
	}
}

func TestGetStatusOK(t *testing.T) {
	setup()

	mux.HandleFunc("/status",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")

			fmt.Fprint(w, `{"account":{"expires":"2014-09-26T19:07:28Z","messages":[],"maxLineups":4,"nextSuggestedConnectTime":"2014-07-29T22:43:22Z"},"lineups":[],"lastDataUpdate":"2014-07-28T14:48:59Z","notifications":[],"systemStatus":[{"date":"2012-12-17T16:24:47Z","status":"Online","details":"All servers running normally."}],"serverID":"serverID1","code":0}`)
		},
	)

	_, err := client.GetStatus("token1")
	if err != nil {
		t.Fatal(err)
	}
}
