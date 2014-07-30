package schedulesdirect

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	baseurl    = "https://json.schedulesdirect.org"
	apiVersion = "/20131021"
)

const (
	sd_err_INVALID_USER = 4003
	// sd_err_NO_HEADENDS  = 4102
)

var (
	err_INVALID_USER = errors.New("Invalid user")
	// err_NO_HEADENDS  = errors.New("No headends")
)

const (
	opLineupAdd = iota
	opLineupDel
)

func hashPassword(password string) string {
	h := sha1.New()
	io.WriteString(h, password)
	return hex.EncodeToString(h.Sum(nil))
}

type tokenRequest struct {
	User string `json:"username"`
	Pass string `json:"password"`
}

type tokenResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	ServerID string `json:"serverID"`
	Token    string `json:"token"`
}

type sdclient struct {
	baseURL string
}

func NewClient() *sdclient {
	return &sdclient{
		baseURL: baseurl,
	}
}

func (c sdclient) GetToken(username, password string) (string, error) {
	tokenReq := tokenRequest{username, hashPassword(password)}

	var buf bytes.Buffer

	errEncode := json.NewEncoder(&buf).Encode(tokenReq)
	if errEncode != nil {
		return "", errEncode
	}

	// TODO: check for something like path.Join() for URLs
	resp, errPost := http.Post(c.baseURL+apiVersion+"/token", "application/json", &buf)
	if errPost != nil {
		return "", errPost
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	var tokenResp tokenResponse

	errDecode := json.NewDecoder(resp.Body).Decode(&tokenResp)
	if errDecode != nil {
		return "", errDecode
	}

	if tokenResp.Code == sd_err_INVALID_USER {
		return "", err_INVALID_USER
	} else if tokenResp.Code != 0 {
		return "", fmt.Errorf("tokenResp.Code != 0: %d", tokenResp.Code)
	}
	if tokenResp.Message != "OK" {
		return "", fmt.Errorf("tokenResp.Message != OK: %s", tokenResp.Message)
	}

	return tokenResp.Token, nil
}

type status struct {
	Account struct {
		Expires                  time.Time `json:"expires"`
		MaxLineups               int       `json:"maxLineups"`
		Messages                 []string  `json:"messages"`
		NextSuggestedConnectTime time.Time `json:"nextSuggestedConnectTime"`
	} `json:"account"`
	Lineups []struct {
		ID       string    `json:"ID"`
		Modified time.Time `json:"modified"`
		Uri      string    `json:"uri"`
	} `json:"lineups"`
	Code           int       `json:"code"`
	LastDataUpdate time.Time `json:"lastDataUpdate"`
	Notifications  []string  `json:"notifications"`
	SystemStatus   []struct {
		Date    time.Time `json:"date"`
		Status  string    `json:"status"`
		Details string    `json:"details"`
	} `json:"systemStatus"`
	ServerID string `json:"serverID"`
}

func (c sdclient) GetStatus(token string) (string, error) {
	var clientHttp http.Client

	req, errNewRequest := http.NewRequest("GET", c.baseURL+apiVersion+"/status", nil)
	if errNewRequest != nil {
		return "", errNewRequest
	}

	req.Header.Add("token", token)

	resp, errDo := clientHttp.Do(req)
	if errDo != nil {
		return "", errDo
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	var s status

	errDecode := json.NewDecoder(resp.Body).Decode(&s)
	if errDecode != nil {
		return "", errDecode
	}

	if s.Code != 0 {
		return "", fmt.Errorf("s.Code != 0: %d", s.Code)
	}

	return "", nil
}

type lineup struct {
	Name string `json:"name"`
	Uri  string `json:"uri"`
}

type headend struct {
	Lineups  []lineup `json:"lineups"`
	Location string   `json:"location"`
	Type     string   `json:"type"`
}

type response struct {
	Response string `json:"response"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	ServerID string `json:"serverID"`
}

type responseAddLineup struct {
	response
	ChangesRemaining int       `json:"changesRemaining"`
	Datetime         time.Time `json:"datetime"`
}

type responseDelLineup struct {
	responseAddLineup
	ChangesRemaining string `json:"changesRemaining"`
}

type lineups struct {
	Datetime time.Time `json:"datetime"`
	Lineups  []struct {
		lineup
		Location string `json:"location"`
		Name     string `json:"name"`
	} `json:"lineups"`
	ServerID string `json:"serverID"`
}

// country must be ISO-3166-1 alpha 3, see : https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3
func (c sdclient) GetHeadends(token, country, postalcode string) (map[string]headend, error) {
	// There's a bug with postal code containing a space
	// https://github.com/SchedulesDirect/JSON-Service/issues/31
	postalcode = strings.Replace(postalcode, " ", "", -1)

	u, err := url.Parse(c.baseURL + apiVersion + "/headends")
	if err != nil {
		return map[string]headend{}, err
	}

	q := u.Query()
	q.Set("country", country)
	q.Set("postalcode", postalcode)
	u.RawQuery = q.Encode()

	var clientHttp http.Client

	req, errNewRequest := http.NewRequest("GET", u.String(), nil)
	if errNewRequest != nil {
		return map[string]headend{}, errNewRequest
	}

	req.Header.Add("token", token)

	resp, errDo := clientHttp.Do(req)
	if errDo != nil {
		return map[string]headend{}, errDo
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return map[string]headend{}, fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	headends := make(map[string]headend)

	data, errRead := ioutil.ReadAll(resp.Body)
	if errRead != nil {
		return map[string]headend{}, errRead
	}

	errUnmarshal := json.Unmarshal(data, &headends)
	if errUnmarshal != nil {
		// when there's an error, the service use another JSON format
		// maybe I should use a struct with both fields
		var respError response

		errUnmarshal2 := json.Unmarshal(data, &respError)
		if errUnmarshal2 != nil {
			return map[string]headend{}, errUnmarshal
		} else {
			return map[string]headend{}, errors.New(respError.Message)
		}
	}

	return headends, nil
}

func addDelLineup(c sdclient, token, uri, method string, typeOpLineup int) (int, error) {
	var clientHttp http.Client

	req, errNewRequest := http.NewRequest(method, c.baseURL+uri, nil)
	if errNewRequest != nil {
		return -1, errNewRequest
	}

	req.Header.Add("token", token)

	resp, errDo := clientHttp.Do(req)
	if errDo != nil {
		return -1, errDo
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return -1, fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	data, errRead := ioutil.ReadAll(resp.Body)
	if errRead != nil {
		return -1, errRead
	}

	var r responseAddLineup

	switch typeOpLineup {
	case opLineupAdd:
		errUnmarshal := json.Unmarshal(data, &r)
		if errUnmarshal != nil {
			return -1, errUnmarshal
		}
	case opLineupDel:
		// ChangesRemaining is a int when adding a lineup and a string when deleting
		// see: https://github.com/SchedulesDirect/JSON-Service/issues/32
		var repDelLineup responseDelLineup

		errUnmarshal := json.Unmarshal(data, &repDelLineup)
		if errUnmarshal != nil {
			return -1, errUnmarshal
		}

		if repDelLineup.Code != 0 {
			return -1, errors.New(repDelLineup.Message)
		}

		r = repDelLineup.responseAddLineup

		var errAtoi error
		r.ChangesRemaining, errAtoi = strconv.Atoi(repDelLineup.ChangesRemaining)
		if errAtoi != nil {
			return -1, errAtoi
		}
	default:
		return -1, fmt.Errorf("typeOpLineup unknown: %d", typeOpLineup)
	}

	if r.Code == 0 && r.Response == "OK" {
		return r.ChangesRemaining, nil
	} else {
		return -1, errors.New(r.Message)
	}
}

func (c sdclient) AddLineup(token, uri string) (int, error) {
	return addDelLineup(c, token, uri, "PUT", opLineupAdd)
}

func (c sdclient) DelLineup(token, uri string) (int, error) {
	return addDelLineup(c, token, uri, "DELETE", opLineupDel)
}

type channelMapping struct {
	Map []struct {
		Channel   string `json:"map"`
		StationId string `json:"stationID"`
	} `json:"map"`
	Metadata struct {
		Lineup    string    `json:"lineup"`
		Modified  time.Time `json:"modified"`
		Transport string    `json:"transport"`
	} `json:"metadata"`
	Stations []struct {
		affiliate   string `json:"affiliate"`
		broadcaster struct {
			city       string `json:"city"`
			country    string `json:"country"`
			postalcode string `json:"postalcode"`
		} `json:"broadcaster"`
		Callsign  string `json:"callsign"`
		Language  string `json:"language"`
		Name      string `json:"name"`
		StationID string `json:"stationID"`
		Logo      struct {
			URL       string `json:"URL"`
			Dimension string `json:"dimension"`
			Md5       string `json:"md5"`
		}
	} `json:"stations"`

	// To catch errors
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c sdclient) GetChannelMapping(token, uri string) (channelMapping, error) {
	var clientHttp http.Client

	req, errNewRequest := http.NewRequest("GET", c.baseURL+uri, nil)
	if errNewRequest != nil {
		return channelMapping{}, errNewRequest
	}

	req.Header.Add("token", token)

	resp, errDo := clientHttp.Do(req)
	if errDo != nil {
		return channelMapping{}, errDo
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return channelMapping{}, fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	var r channelMapping

	errDecode := json.NewDecoder(resp.Body).Decode(&r)
	if errDecode != nil {
		return channelMapping{}, errDecode
	}

	if r.Code != 0 {
		return channelMapping{}, errors.New(r.Message)
	}

	return r, nil
}

func (c sdclient) GetLineups(token string) (lineups, error) {
	var clientHttp http.Client

	req, errNewRequest := http.NewRequest("GET", c.baseURL+apiVersion+"/lineups", nil)
	if errNewRequest != nil {
		return lineups{}, errNewRequest
	}

	req.Header.Add("token", token)

	resp, errDo := clientHttp.Do(req)
	if errDo != nil {
		return lineups{}, errDo
	}
	defer resp.Body.Close()

	// TODO: only expect 400 for error code 4102
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return lineups{}, fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	data, errRead := ioutil.ReadAll(resp.Body)
	if errRead != nil {
		return lineups{}, errRead
	}

	var r response

	errUnmarshal := json.Unmarshal(data, &r)
	if errUnmarshal != nil {
		return lineups{}, errUnmarshal
	} else if r.Message != "" {
		return lineups{}, errors.New(r.Message)
	} else {
		var l lineups

		errUnmarshal2 := json.Unmarshal(data, &l)
		if errUnmarshal2 != nil {
			return lineups{}, errUnmarshal
		}

		return l, nil
	}
}
