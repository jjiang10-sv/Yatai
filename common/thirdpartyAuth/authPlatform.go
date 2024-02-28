package thirdpartyAuth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	url2 "net/url"

	"github.com/bentoml/yatai/api-server/config"
	"github.com/bentoml/yatai/common/clients"
	"github.com/bentoml/yatai/common/utils/errorx"
	"github.com/bentoml/yatai/common/utils/metrics"
)

const (
	DOMAIN_NAME = "http://172.16.1.97:32763"
	PREFIX_API  = "/api/v1/"
)

var (
	instanceAuthClient *AuthClient
)

// AuthRedirectParams defines parameters for AuthRedirect.
type AuthClient struct {
	client *metrics.HTTPClient
}

// {
// 	"data"：{
// 	"access_token"："c398bc9e-b82f-47a4-89a0-e17108e9c71a"，
// 	"expires_in"：7200，
// 	"refresh_token"："c438058125d346fc8d24a35a25f34cd4"，
// 	"token_type"："bearer"
// 	}，

type AssesTokenData struct {
	Data TokenContainer `json:"data"`
}

type TokenContainer struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    uint16 `json:"expires_in"`
	Bearer       string `json:"token_type"`
}

type AccessTokenQuery struct {
	ClientId     string `url:"client_id,omitempty"`
	ClientSecret string `url:"client_secret,omitempty"`
}

func NewSingleAuthClient() *AuthClient {
	once.Do(func() {
		// todo handle error
		metricsHttpClient, _ := clients.SetupHttpClient("", "")
		res := AuthClient{
			client: metricsHttpClient,
		}
		instanceAuthClient = &res
	})
	return instanceAuthClient

}

func (a *AuthClient) GetAssesToken(ctx context.Context) (tokenData *TokenContainer, err error) {

	authConfig := config.YataiConfig.Oauth2
	params := AccessTokenQuery{
		ClientId:     authConfig.ClientID,
		ClientSecret: authConfig.ClientSecret,
	}

	urlQuery, err := clients.QueryFromInterface(params)
	if err != nil {
		return
	}
	url, err := url2.ParseRequestURI(DOMAIN_NAME + PREFIX_API)
	if err != nil {
		return
	}

	req, err := clients.NewRequestWithPromContextSpecial(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s://%s", url.Scheme, url.Host),
		url.Path,
		nil,
		urlQuery,
		nil,
	)
	if err != nil {
		return
	}
	// req.Header.Add(`Authorization`, token)
	resp, err := a.client.Do(req)
	if err != nil {
		err = errorx.Errorf("Unable to retrieve critical resource --(Error)--> [[ %v ]]", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = errorx.Errorf("Received status code (%d) while retrieving token", resp.StatusCode)
		return
	}

	var tokenRes AssesTokenData
	if e := json.NewDecoder(resp.Body).Decode(&tokenRes); e != nil {
		err = errorx.Errorf("Unable to read JSON Auth resource --(Error)--> [[ %v ]]", e)
		return
	}
	return &tokenRes.Data, nil
}
