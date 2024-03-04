package thirdpartyauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"sync"

	"github.com/bentoml/yatai/api-server/config"
	"github.com/bentoml/yatai/common/clients"
	"github.com/bentoml/yatai/common/consts"
	"github.com/bentoml/yatai/common/utils/errorx"
	"github.com/bentoml/yatai/common/utils/metrics"
	"github.com/gin-gonic/gin"
)

const (
	DOMAIN_NAME = "http://172.16.1.97:32763"
	PREFIX_API  = "/api/v1/"
)

var (
	instanceAuthClient *authClient
	onceAuthClient     sync.Once
)

type AuthClient interface {
	GetAssesToken(ctx *gin.Context) (*TokenContainer, error)
	RefreshAssesToken(ctx *gin.Context, refreshToken string) (*TokenContainer, error)
	GetUserToken(ctx *gin.Context, code, accessToken string) (*UserTokenParams, error)
	GetUserApiPermissions(ctx *gin.Context) (*[]AllowedApiList, error)
}

// AuthRedirectParams defines parameters for AuthRedirect.
type authClient struct {
	client *metrics.HTTPClient
}

type AssesTokenData struct {
	Data     TokenContainer `json:"data"`
	ErrCode  uint8          `json:"error_code"`
	ErrorMsg string         `json:"error_msg"`
}

type TokenContainer struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type AccessTokenQuery struct {
	ClientId     string `url:"client_id,omitempty"`
	ClientSecret string `url:"client_secret,omitempty"`
}

type RefreshAccessTokenQuery struct {
	AccessTokenQuery
	RefreshToken string `url:"refresh_token,omitempty"`
}
type ThirdPartyLoginAuthCode struct {
	// code for access token exchange
	Code string `json:"code"`
}

type GetClientsApiBodyParams struct {
	// code for access token exchange
	ClientId  string `json:"client_id"`
	UserToken string `json:"user_token"`
}

type ThirdPartyLoginAccessCode struct {
	AccessToken string `url:"access_token,omitempty"`
}

type ThirdPartyLoginUserTokenRespsone struct {
	Data     UserTokenParams `json:"data"`
	ErrCode  uint16          `json:"error_code"`
	ErrorMsg string          `json:"error_msg"`
}

type UserTokenParams struct {
	UserToken string `json:"user_token"`
}



// data"：
// [{
// ""：1，//安全级别
// ""："GET"，//请求方式POSTGETPUT
// ""："home"，//api名称
// ""："/index"，//api路由
// ""："xxxxxx"，//api编码
// ""："0"，//父菜单编码
// ""："http：//127.0.0.1"，//api域名
// ""：9，//类型(9：接口)
// ""："xxxxxx"//子平台clientid
// }]，


type GetUserApiPermissionsResponse struct {
	Data     []AllowedApiList `json:"data"`
	ErrCode  uint16          `json:"error_code"`
	ErrorMsg string          `json:"error_msg"`
}

type AllowedApiList struct {
	SecurityLevel uint `json:"securityLevel"`
	ApiMethod string `json:"apiMethod"`
	ApiName string `json:"apiName"`
	ApiRouter string `json:"apiRouter"`
	ApiCode string `json:"apiCode"`
	MenuParent string `json:"menuParent"`
	ApiDomain string `json:"apiDomain"`
	MenuType uint `json:"menuType"`
	ClientId string `json:"client_id"`
}

type GetUserMenuPermissionsResponse struct {
	Data     []AllowedMenu `json:"data"`
	ErrCode  uint16          `json:"error_code"`
	ErrorMsg string          `json:"error_msg"`
}
type AllowedMenu struct {
	MenuCode uint `json:"menuCode"`
	MenuRouter string `json:"menuRouter"`
	MenuName string `json:"menuName"`
	MenuDomain string `json:"menuDomain"`
	MenuParent string `json:"menuParent"`
	MenuIcon string `json:"menuIcon"`
	MenuType uint `json:"menuType"`
	ClientId string `json:"client_id"`
}
// ""：""，
// ""：""，
// ""：false，
// ""：""，
// ""：0，
// ""：1，
// ""：""，
// ""：""


type GetUserInfoDetailsResponse struct {
	Data     UserDetails `json:"data"`
	ErrCode  uint16          `json:"error_code"`
	ErrorMsg string          `json:"error_msg"`
}
type UserDetails struct {
	IamOpenId uint `json:"iam_openid"`
	PrMobile string `json:"prMobile"`
	IsFrozen string `json:"isFrozen"`
	PrUserName string `json:"prUserName"`
	PrStatus string `json:"prStatus"`
	PrSex string `json:"prSex"`
	PrUserEmail uint `json:"prUserEmail"`
	PrPinyin string `json:"prPinyin"`
}

func NewSingleAuthClient() *authClient {
	onceAuthClient.Do(func() {
		// todo handle error
		metricsHttpClient, _ := clients.SetupHttpClient("", "")
		res := authClient{
			client: metricsHttpClient,
		}
		instanceAuthClient = &res
	})
	return instanceAuthClient

}

func (a *authClient) GetAssesToken(ctx *gin.Context) (*TokenContainer, error) {

	authConfig := config.YataiConfig.Oauth2
	params := AccessTokenQuery{
		ClientId:     authConfig.ClientID,
		ClientSecret: authConfig.ClientSecret,
	}

	body, err := httpGet(params, "oauth/token", " Refresh AccessToken", ctx, a.client)
	if err != nil {
		return nil, err
	}
	var tokenRes AssesTokenData
	if e := json.NewDecoder(body).Decode(&tokenRes); e != nil {
		err = errorx.Errorf("Unable to read JSON Auth resource --(Error)--> [[ %v ]]", e)
		return nil, err
	}
	return &tokenRes.Data, nil
}

func (a *authClient) RefreshAssesToken(ctx *gin.Context, refreshToken string) (*TokenContainer, error) {

	authConfig := config.YataiConfig.Oauth2
	params := RefreshAccessTokenQuery{
		AccessTokenQuery: AccessTokenQuery{
			ClientId:     authConfig.ClientID,
			ClientSecret: authConfig.ClientSecret,
		},
		RefreshToken: refreshToken,
	}

	body, err := httpGet(params, "oauth/token", " Refresh AccessToken", ctx, a.client)
	if err != nil {
		return nil, err
	}
	var tokenRes AssesTokenData
	if e := json.NewDecoder(body).Decode(&tokenRes); e != nil {
		err = errorx.Errorf("Unable to read JSON Auth resource --(Error)--> [[ %v ]]", e)
		return nil, err
	}
	return &tokenRes.Data, nil
}

func (a *authClient) GetUserToken(ctx *gin.Context, code, accessToken string) (*UserTokenParams, error) {

	queries := ThirdPartyLoginAccessCode{AccessToken: accessToken}
	body := ThirdPartyLoginAuthCode{Code: code}

	body_, err := httpPost(queries, body, "user/get_usertoken", " Refresh AccessToken", ctx, a.client)
	if err != nil {
		return nil, err
	}
	var tokenRes ThirdPartyLoginUserTokenRespsone
	if e := json.NewDecoder(body_).Decode(&tokenRes); e != nil {
		err = errorx.Errorf("get User token failed")
		return nil, err
	}
	return &tokenRes.Data, nil
	// if tokenRes.ErrCode != 0 {
	// 	return nil, errorx.Errorf("Unable to get user token error msg : %s ==== error code : %d", tokenRes.ErrorMsg, tokenRes.ErrCode)
	// }
	// if res, ok := tokenRes.Data.(UserTokenParams); !ok {
	// 	return nil, errorx.Errorf("Unable to convert to userTokenParams data structure")
	// }else {
	// 	return &tokenRes, nil
	// }

}

func (a *authClient) GetUserApiPermissions(ctx *gin.Context) (*[]AllowedApiList, error) {

	queries := ThirdPartyLoginAccessCode{AccessToken: ctx.GetHeader(consts.HeaderAccessToken)}
	body := GetClientsApiBodyParams{
		ClientId:  ctx.GetHeader(config.YataiConfig.Oauth2.ClientID),
		UserToken: ctx.GetHeader(consts.YataiUserTokenHeaderName),
	}

	body_, err := httpPost(queries, body, "user/get_client_apis", " Get User Api Permissions", ctx, a.client)
	if err != nil {
		return nil, err
	}
	var res GetUserApiPermissionsResponse
	if e := json.NewDecoder(body_).Decode(&res); e != nil {
		err = errorx.Errorf("get user api permissions failed")
		return nil, err
	}
	return &res.Data,nil
}


func (a *authClient) GetUserMenuPermissions(ctx *gin.Context) (*[]AllowedMenu, error) {

	queries := ThirdPartyLoginAccessCode{AccessToken: ctx.GetHeader(consts.HeaderAccessToken)}
	body := GetClientsApiBodyParams{
		ClientId:  ctx.GetHeader(config.YataiConfig.Oauth2.ClientID),
		UserToken: ctx.GetHeader(consts.YataiUserTokenHeaderName),
	}

	body_, err := httpPost(queries, body, "user/get_client_menus", " Get User Api Permissions", ctx, a.client)
	if err != nil {
		return nil, err
	}
	var res GetUserMenuPermissionsResponse
	if e := json.NewDecoder(body_).Decode(&res); e != nil {
		err = errorx.Errorf("get user api permissions failed")
		return nil, err
	}
	return &res.Data,nil
}


func (a *authClient) GetUserInfoDetails(ctx *gin.Context) (*UserDetails, error) {

	queries := ThirdPartyLoginAccessCode{AccessToken: ctx.GetHeader(consts.HeaderAccessToken)}
	body := GetClientsApiBodyParams{
		// ClientId:  ctx.GetHeader(config.YataiConfig.Oauth2.ClientID),
		UserToken: ctx.GetHeader(consts.YataiUserTokenHeaderName),
	}

	body_, err := httpPost(queries, body, "user/get_client_menus", " Get User Api Permissions", ctx, a.client)
	if err != nil {
		return nil, err
	}
	var res GetUserInfoDetailsResponse
	if e := json.NewDecoder(body_).Decode(&res); e != nil {
		err = errorx.Errorf("get user api permissions failed")
		return nil, err
	}
	return &res.Data,nil
}

func httpGet(params interface{}, reqPath string, msg string, ctx context.Context, client *metrics.HTTPClient) (io.ReadCloser, error) {
	urlQuery, err := clients.QueryFromInterface(params)
	if err != nil {
		return nil, err
	}
	url, err := url2.ParseRequestURI(DOMAIN_NAME + PREFIX_API + reqPath)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	// req.Header.Add(`Authorization`, token)
	resp, err := client.Do(req)
	if err != nil {
		err = errorx.Errorf("Unable to %s --(Error)--> [[ %v ]]", msg, err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = errorx.Errorf("Received status code (%d) while %s", resp.StatusCode, msg)
		return nil, err
	}
	return resp.Body, nil
}

func httpPost(queries, body interface{}, reqPath string, msg string, ctx context.Context, client *metrics.HTTPClient) (io.ReadCloser, error) {
	var urlQuery map[string]interface{}
	var err error
	if queries != nil {
		urlQuery, err = clients.QueryFromInterface(queries)
		if err != nil {
			return nil, err
		}
	}
	url, err := url2.ParseRequestURI(DOMAIN_NAME + PREFIX_API + reqPath)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, errorx.Errorf("ExchangeSiteWcControlPolicyRequest: Unable to serialize request: %s", err.Error())
	}
	body_ := bytes.NewBuffer(jsonBytes)

	req, err := clients.NewRequestWithPromContextSpecial(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s://%s", url.Scheme, url.Host),
		url.Path,
		nil,
		urlQuery,
		body_,
	)
	if err != nil {
		return nil, err
	}
	// req.Header.Add(`Authorization`, token)
	resp, err := client.Do(req)
	if err != nil {
		err = errorx.Errorf("Unable to %s --(Error)--> [[ %v ]]", msg, err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = errorx.Errorf("Received status code (%d) while %s", resp.StatusCode, msg)
		return nil, err
	}
	return resp.Body, nil
}

////domain.com/api/v1/oauth/token?client_id=CLIENT_ID&client_secret=CLIENT_SECRET&refresh_token=REFRESH_TOKEN
