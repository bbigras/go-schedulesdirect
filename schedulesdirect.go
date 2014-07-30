package schedulesdirect

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseurl = "https://json.schedulesdirect.org/20131021"
)

const (
	sd_err_INVALID_USER = 4003
)

var (
	err_INVALID_USER = errors.New("Invalid user")
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
	resp, errPost := http.Post(c.baseURL+"/token", "application/json", &buf)
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

	req, errNewRequest := http.NewRequest("GET", c.baseURL+"/status", nil)
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

// country must be ISO-3166-1 alpha 3, see : https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3
func (c sdclient) GetHeadends(token, country, postalcode string) (map[string]headend, error) {
	// There's a bug with postal code containing a space
	// https://github.com/SchedulesDirect/JSON-Service/issues/31
	postalcode = strings.Replace(postalcode, " ", "", -1)

	u, err := url.Parse(c.baseURL + "/headends")
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

	errDecode := json.NewDecoder(resp.Body).Decode(&headends)
	if errDecode != nil {
		return map[string]headend{}, errDecode
	}

	return headends, nil
}
