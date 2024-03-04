package thirdpartyauth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"sync"

	"encoding/pem"
	"fmt"
	"strings"

	"github.com/bentoml/yatai-schemas/schemasv1"
	"github.com/bentoml/yatai/api-server/config"
	"github.com/bentoml/yatai/common/utils/errorx"
	"github.com/dgrijalva/jwt-go/v4"
	"golang.org/x/oauth2"
)

var (
	instance *OAuth2
	once     sync.Once
)

// AuthRedirectParams defines parameters for AuthRedirect.
type AuthRedirectParams struct {
	// frontend jump back redirect url in base64
	FeReturn string `json:"feReturn"`
}

type OAuth2 struct {
	config.Oauth2Config
	rsaPublicKey *rsa.PublicKey
}

// AuthParams defines parameters for Auth.
type AuthParams struct {
	// code for access token exchange
	Code *string `query:"code"`
	// state from sso
	State *string     `query:"state"`
	Q     schemasv1.Q `query:"q"`
}

// ThirdPartyLogin defines parameters for ThirdPartyLogin.
type ThirdPartyLogin struct {
	// code for access token exchange
	Code *string `query:"code"`
}


// AuthParams defines parameters for Auth.
// type AuthParams struct {
// 	// code for access token exchange
// 	Code string `json:"code"`
// 	// state from sso
// 	State string `json:"state"`
// }

var (
	// oidc.ScopeOpenID "microprofile-jwt", "profile", "phone", "address", "email" ", "https://graph.microsoft.com/user.read""
	// if specified with openid email , then use graph api which can not be validated
	scopes = []string{"258f23c3-2ead-4f9f-ac71-c24bcd889060/.default"}
)

func (oa *OAuth2) ResolveRsaPublicKey() *OAuth2 {
	// if rsaPub, err := parseRSAPublicKey(oa.PublicKey); err != nil {
	// 	appCtx.Fatal("Failed to parse rsaPublicKey")
	// } else {
	// 	oa.rsaPublicKey = rsaPub
	// }
	return oa
}

func parseRSAPublicKey(publicKey string) (*rsa.PublicKey, error) {
	data, _ := pem.Decode([]byte(strings.Trim(publicKey, " ")))
	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)
	if err != nil {
		return nil, err
	}
	if rsaPub, ok := publicKeyImported.(*rsa.PublicKey); ok {
		return rsaPub, nil
	}
	return nil, fmt.Errorf("%+v is not a *rsa.PublicKey", publicKeyImported)
}



func NewSingleOAuth2() *OAuth2 {
	once.Do(func() {
		oa := &OAuth2{}
		oa.Oauth2Config = config.YataiConfig.Oauth2
		oa.ResolveRsaPublicKey()

		instance = oa
	})
	return instance

}

func (oa *OAuth2) GetConfig(returnURL string) (*oauth2.Config, error) {
	// provider, err := oidc.NewProvider(context.Background(), oa.SSOIssuer)
	// if err != nil {
	// 	return nil, err
	// }
	tenantId := "9026c5f4-86d0-4b9f-bd39-b7d4d0fb4674"
	tokenUrl := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantId)
	authUrl := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantId)
	return &oauth2.Config{
		ClientID:     oa.ClientID,
		ClientSecret: oa.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authUrl,
			TokenURL:  tokenUrl,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: returnURL,
		Scopes:      scopes,
	}, nil
}

func (oa *OAuth2) AuthCodeURL(state, returnURL string) (string, error) {
	config, err := oa.GetConfig(returnURL)
	if err != nil {
		return "", err
	}
	return config.AuthCodeURL(state), nil
}

// Exchange
func (oa *OAuth2) Exchange(code, returnURL string) (*oauth2.Token, error) {
	config, err := oa.GetConfig(returnURL)
	if err != nil {
		return nil, err
	}
	// jsonstring, _ := json.Marshal(&config)
	// fmt.Println("=========", string(jsonstring))
	token, err := config.Exchange(context.Background(), code, oauth2.AccessTypeOffline)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (oa *OAuth2) GetClaims(accessToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return oa.rsaPublicKey, nil
	}, jwt.WithoutAudienceValidation())
	if err != nil {
		switch err.(type) {
		case *jwt.TokenExpiredError:
			return nil, errorx.New("token expired")
		case *jwt.TokenNotValidYetError:
			return nil, errorx.New("token not valid yet")
		case *jwt.InvalidIssuerError:
			return nil, errorx.New("token issuer is invalid")
		default:
			return nil, errorx.WithStack(err)
		}
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func GetTokenRefreshExpiresIn(token *oauth2.Token) int64 {
	v := token.Extra("refresh_expires_in")
	if v == nil {
		return 0
	}
	switch vv := v.(type) {
	case int64:
		return vv
	case float64:
		return int64(vv)
	}

	return 0
}
