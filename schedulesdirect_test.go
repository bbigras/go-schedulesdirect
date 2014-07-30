package schedulesdirect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func testUrlParameter(t *testing.T, r *http.Request, parameter, expectedValue string) {
	if parameter == "postalcode" {
		// There's a bug with postal code containing a space
		// https://github.com/SchedulesDirect/JSON-Service/issues/31
		expectedValue = strings.Replace(expectedValue, " ", "", -1)
	}

	p := r.URL.Query().Get(parameter)

	if p != expectedValue {
		t.Fatalf("parameter (%s (%s)) != expectedValue (%s)", parameter, p, expectedValue)
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

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client.baseURL != baseurl {
		t.Fail()
	}
}

func TestHashPassword(t *testing.T) {
	if hashPassword("testpassword") != "8bb6118f8fd6935ad0876a3be34a717d32708ffd" {
		t.Fail()
	}

}

func TestGetTokenInvalidUser(t *testing.T) {
	setup()

	mux.HandleFunc("/token",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")

			var tokenReq tokenRequest

			errDecode := json.NewDecoder(r.Body).Decode(&tokenReq)
			if errDecode != nil {
				t.Fatal(errDecode)
			}

			fmt.Fprint(w, `{"response":"INVALID_USER","code":4003,"serverID":"serverID1","message":"Invalid user.","datetime":"2014-07-29T01:00:28Z"}`)
		},
	)

	_, errToken := client.GetToken("user1", "pass1")
	if errToken != err_INVALID_USER {
		t.Fatalf("errToken != err_INVALID_USER (%s)", errToken.Error())
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

func TestGetHeadendsOK(t *testing.T) {
	setup()

	mux.HandleFunc("/headends",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			testUrlParameter(t, r, "country", "CAN")
			testUrlParameter(t, r, "postalcode", "H0H 0H0")

			fmt.Fprint(w, `{"0000001":{"lineups":[{"name":"name1","uri":"uri1"},{"name":"name2","uri":"uri2"}],"location":"City1","type":"type1"},"0000002":{"lineups":[{"name":"name3","uri":"uri3"}],"location":"City2","type":"type2"}}`)
		},
	)

	headends, errGetHeadends := client.GetHeadends("token1", "CAN", "H0H 0H0")
	if errGetHeadends != nil {
		t.Fatal(errGetHeadends)
	}

	if len(headends) != 2 {
		t.Fatalf("len(headends) != 2: %d", len(headends))
	} else {
		if len(headends["0000001"].Lineups) != 2 {
			t.Fatalf(`len(headends["0000001"].Lineups) != 2: %d`, len(headends["0000001"].Lineups))
		} else if headends["0000001"].Lineups[0].Name != "name1" {
			t.Fatalf(`headends["0000001"].Lineups[0].Name != "name1": %s`, headends["0000001"].Lineups[0].Name)
		}
		if len(headends["0000002"].Lineups) != 1 {
			t.Fatalf(`len(headends["0000002"].Lineups) != 1: %d`, len(headends["0000002"].Lineups))
		}
	}
}

func TestGetHeadendsFailsWithMessage(t *testing.T) {
	setup()

	mux.HandleFunc("/headends",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			testUrlParameter(t, r, "country", "CAN")
			testUrlParameter(t, r, "postalcode", "H0H 0H0")

			fmt.Fprint(w, `{"response":"INVALID_PARAMETER:COUNTRY","code":2050,"serverID":"serverID1","message":"The COUNTRY parameter must be ISO-3166-1 alpha 3. See http:\/\/en.wikipedia.org\/wiki\/ISO_3166-1_alpha-3","datetime":"2014-07-29T23:16:52Z"}`)
		},
	)

	_, errGetHeadends := client.GetHeadends("token1", "CAN", "H0H 0H0")
	if errGetHeadends.Error() != "The COUNTRY parameter must be ISO-3166-1 alpha 3. See http://en.wikipedia.org/wiki/ISO_3166-1_alpha-3" {
		t.Fail()
	}
}

func TestGetHeadendsFailsWithMessage2(t *testing.T) {
	setup()

	mux.HandleFunc("/headends",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			testUrlParameter(t, r, "country", "CAN")
			testUrlParameter(t, r, "postalcode", "H0H 0H0")

			fmt.Fprint(w, `{"response":"REQUIRED_PARAMETER_MISSING:COUNTRY","code":2004,"serverID":"serverID1","message":"In order to search for lineups, you must supply a 3-letter country parameter.","datetime":"2014-07-29T23:15:18Z"}`)
		},
	)

	_, errGetHeadends := client.GetHeadends("token1", "CAN", "H0H 0H0")
	if errGetHeadends.Error() != "In order to search for lineups, you must supply a 3-letter country parameter." {
		t.Fail()
	}
}
