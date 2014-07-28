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

type TokenRequest struct {
	User string `json:"username"`
	Pass string `json:"password"`
}

type TokenResponse struct {
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
	tokenRequest := TokenRequest{username, hashPassword(password)}

	data, errM := json.Marshal(tokenRequest)
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

	var repToken TokenResponse

	errUnmarshal := json.Unmarshal(r, &repToken)
	if errUnmarshal != nil {
		return "", errUnmarshal
	}

	if repToken.Code != 0 {
		return "", fmt.Errorf("repToken.Code != 0: %d", repToken.Code)
	}
	if repToken.Message != "OK" {
		return "", fmt.Errorf("repToken.Message != OK: %s", repToken.Message)
	}

	return repToken.Token, nil
}
