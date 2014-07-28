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
		baseUrl: server.URL,
	}
}

func testMethod(t *testing.T, r *http.Request, expectedMethod string) {
	if r.Method != expectedMethod {
		t.Fatalf("method (%s) != expectedMethod (%s)", r.Method, expectedMethod)
	}

	var tokenReq tokenRequest

	errDecode := json.NewDecoder(r.Body).Decode(&tokenReq)
	if errDecode != nil {
		t.Fatal(errDecode)
	}
}

func TestGetTokenOK(t *testing.T) {
	setup()

	mux.HandleFunc("/token",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")
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
