package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/bentoml/yatai/common/thirdpartyAuth"
	"github.com/bentoml/yatai/common/utils/bizerr"
	"github.com/bentoml/yatai/common/utils/errorx"
	"github.com/gin-gonic/gin"

	"github.com/bentoml/yatai/common/utils/logx"
	"github.com/bentoml/yatai/common/utils/xidx"
	//"github.com/bentoml/yatai/common/clients"
)

type ReqContext struct {
	context.Context
	xid        string
	GinContext *gin.Context
	logx.ReqLogger
}

func NewReqContext(ctx context.Context) ReqContext {
	rc := ReqContext{
		Context:    ctx,
		xid:        xidx.GenXid(),
		GinContext: &gin.Context{},
	}
	rc.ReqLogger = logx.MustNewReqLogger(rc.xid)
	return rc
}

type thirdPartyAuthService struct {
	thirdpartyAuth.OAuth2
	thirdpartyAuth.AuthClient
	ReqContext
}

func NewThirdPartyAuthService(ctx context.Context) *thirdPartyAuthService {
	svc := &thirdPartyAuthService{
		OAuth2:     *thirdpartyAuth.NewSingleOAuth2(),
		AuthClient: *thirdpartyAuth.NewSingleAuthClient(),
		ReqContext: NewReqContext(ctx),
	}

	return svc
}

func (o *thirdPartyAuthService) Auth(params *thirdpartyAuth.AuthParams) (interface{}, error) {
	oauth2Token, err := o.Exchange(*params.Code, returnUrl())
	if err != nil {
		o.Error("can't exchange token", "error", err)
		return nil, errorx.Wrap(err, "can't exchange token")
	}
	//claims, err := l.GetClaims(oauth2Token.AccessToken)
	claims, err := o.GetClaims(oauth2Token.AccessToken)
	if err != nil {
		o.Error("can't exchange token", "error", err)
		return nil, errorx.Wrap(err, "can't get claims from access token")
	}
	email := claims["email"]
	o.Info("user logged in", "email", email)

	feReturn, err := base64.StdEncoding.DecodeString(*params.State)
	if err != nil {
		return nil, bizerr.New(http.StatusBadRequest, errorx.Wrap(err, "feReturn url not valid base64 encoded"))
	}
	o.GinContext.Redirect(http.StatusFound, string(feReturn))
	return nil, nil
}

func (t thirdPartyAuthService) AuthRedirect(params thirdpartyAuth.AuthRedirectParams) (interface{}, error) {
	url, err := t.AuthCodeURL(params.FeReturn, returnUrl())
	if err != nil {
		return "", errorx.Wrap(err, "can't get redirect url")
	}
	//l.ReqCtx.Gin().Redirect(http.StatusFound, url)
	return url, nil
}

func (t thirdPartyAuthService) ThirdPartyLogin(ctx context.Context) (*thirdpartyAuth.TokenContainer, error) {

	return t.AuthClient.GetAssesToken(ctx)

}

func returnUrl() string {
	return fmt.Sprintf("%s%s/auth", "l.AppCtx.APIEndpoint()", "l.AppCtx.WhoAmI()")
}
