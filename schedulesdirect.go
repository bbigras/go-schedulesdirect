package schedulesdirect

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	BASEURL = "https://json.schedulesdirect.org/20131021"
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
	baseUrl string
}

func NewClient() *sdclient {
	return &sdclient{
		baseUrl: BASEURL,
	}
}

func (c sdclient) GetToken(username, password string) (string, error) {
	tokenReq := tokenRequest{username, hashPassword(password)}

	data, errM := json.Marshal(tokenReq)
	if errM != nil {
		return "", errM
	}

	reader := bytes.NewReader(data)

	// TODO: check for something like path.Join() for URLs
	resp, errPost := http.Post(c.baseUrl+"/token", "application/json", reader)
	if errPost != nil {
		return "", errPost
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("resp.StatusCode != 200: %d", resp.StatusCode)
	}

	r, errRead := ioutil.ReadAll(resp.Body)
	if errRead != nil {
		return "", errRead
	}

	var tokenResp tokenResponse

	errUnmarshal := json.Unmarshal(r, &tokenResp)
	if errUnmarshal != nil {
		return "", errUnmarshal
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("tokenResp.Code != 0: %d", tokenResp.Code)
	}
	if tokenResp.Message != "OK" {
		return "", fmt.Errorf("tokenResp.Message != OK: %s", tokenResp.Message)
	}

	return tokenResp.Token, nil
}
