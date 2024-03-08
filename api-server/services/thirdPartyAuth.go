package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/bentoml/yatai-schemas/modelschemas"
	"github.com/bentoml/yatai/api-server/models"
	"github.com/bentoml/yatai/common/consts"
	thirdpartyauth "github.com/bentoml/yatai/common/thirdpartyAuth"
	"github.com/bentoml/yatai/common/utils/bizerr"
	"github.com/bentoml/yatai/common/utils/cache"
	"github.com/bentoml/yatai/common/utils/errorx"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/bentoml/yatai/common/utils/logx"
	"github.com/bentoml/yatai/common/utils/xidx"
	//"github.com/bentoml/yatai/common/clients"
)

type ReqContext struct {
	xid        string
	GinContext *gin.Context
	logx.ReqLogger
}

func NewReqContext(ctx *gin.Context) ReqContext {
	rc := ReqContext{

		xid:        xidx.GenXid(),
		GinContext: ctx,
	}
	rc.ReqLogger = logx.MustNewReqLogger(rc.xid)
	return rc
}

type thirdPartyAuthService struct {
	thirdpartyauth.OAuth2
	thirdpartyauth.AuthClient
	ReqContext
}

func NewThirdPartyAuthService(ctx *gin.Context) *thirdPartyAuthService {
	a := thirdpartyauth.NewSingleOAuth2()
	b := thirdpartyauth.NewSingleAuthClient()
	svc := &thirdPartyAuthService{
		OAuth2:     *a,
		AuthClient: b,
		ReqContext: NewReqContext(ctx),
	}

	return svc
}

// func (t *thirdPartyAuthService) Create(ctx context.Context, opt *thirdpartyauth.TokenContainer) (*models.AccessToken, error) {
// 	accessToken := models.AccessToken{
// 		AccessToken:  opt.AccessToken,
// 		RefreshToken: opt.RefreshToken,
// 		ExpiredIn:    opt.ExpiresIn,
// 	}
// 	err := mustGetSession(ctx).Create(&accessToken).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &accessToken, err
// }

func (s *apiTokenService) GetAccessByTokenId(ctx context.Context, accessTokenId, userId uint, name string) (*models.AccessToken, error) {
	var accessToken models.AccessToken
	err := getBaseQuery(ctx, s).Where("id = ?", accessTokenId).First(&accessToken).Error
	if err != nil {
		return nil, err
	}
	if accessToken.ID == 0 {
		return nil, consts.ErrNotFound
	}
	return &accessToken, nil
}

func (*thirdPartyAuthService) getBaseDB(ctx context.Context) *gorm.DB {
	return mustGetSession(ctx).Model(&models.AccessToken{})
}

func (o *thirdPartyAuthService) Auth(params *thirdpartyauth.AuthParams) (interface{}, error) {
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

func (t thirdPartyAuthService) AuthRedirect(params thirdpartyauth.AuthRedirectParams) (interface{}, error) {
	url, err := t.AuthCodeURL(params.FeReturn, returnUrl())
	if err != nil {
		return "", errorx.Wrap(err, "can't get redirect url")
	}
	//l.ReqCtx.Gin().Redirect(http.StatusFound, url)
	return url, nil
}


func (t thirdPartyAuthService) ThirdPartyLogin(params *thirdpartyauth.ThirdPartyLogin) (*thirdpartyauth.UserInfoAndMenus, error) {

	accessToken, err := t.GetAssesToken(t.GinContext)
	if err != nil {
		return nil, err
	}
	t.GinContext.Header(consts.HeaderAccessToken, accessToken.AccessToken)
	t.GinContext.Header(consts.HeaderRefreshToken, accessToken.RefreshToken)
	t.GinContext.Header(consts.HeaderTokenExpiredIn, fmt.Sprint(accessToken.ExpiresIn))
	t.GinContext.Header(consts.HeaderTokenType, accessToken.TokenType)

	userToken, err := t.GetUserToken(t.GinContext, *params.Code, accessToken.AccessToken)
	if err != nil {
		return nil, err
	}
	t.GinContext.Header(consts.YataiUserTokenHeaderName, userToken.UserToken)
	// register user
	userInfo, err := t.GetUserInfoDetails(t.GinContext, accessToken.AccessToken, userToken.UserToken)
	if err != nil {
		return nil, err
	}
	userInfo.PrUserName = "tester-0305"
	existUser, err := UserService.GetByName(t.GinContext, userInfo.PrUserName)
	if err != nil {
		if err.Error() != "record not found" {
			return nil, err
		}
	}
	if existUser == nil {
		user, err := UserService.Create(t.GinContext, CreateUserOption{
			Name:      userInfo.PrUserName,
			FirstName: userInfo.PrUserName,
			//LastName:  userInfo.PrUserName,
			Perm: modelschemas.UserPermPtr(modelschemas.UserPermAdmin),
		})
		if err != nil {
			return nil, err
		}

		currentUser, err := UserService.GetByName(t.GinContext, "admin")
		if err != nil {
			return nil, errors.Wrap(err, "get current user")
		}
		org, err := OrganizationService.GetByName(t.GinContext, "default")
		if err != nil {
			return nil, errors.Wrap(err, "get organization")
		}
		// if err = OrganizationController.canOperate(ctx, org); err != nil {
		// 	return nil, err
		// }

		_, err = OrganizationMemberService.Create(t.GinContext, currentUser.ID, CreateOrganizationMemberOption{
			CreatorId:      currentUser.ID,
			UserId:         user.ID,
			OrganizationId: org.ID,
			Role:           modelschemas.MemberRoleAdmin,
		})
		if err != nil {
			return nil, errors.Wrap(err, "create organization member")
		}

		majorCluster, err := OrganizationService.GetMajorCluster(t.GinContext, org)
		if err != nil {
			return nil, errors.Wrap(err, "get major cluster")
		}

		clusterRole := modelschemas.MemberRoleAdmin

		_, err = ClusterMemberService.Create(t.GinContext, currentUser.ID, CreateClusterMemberOption{
			CreatorId: currentUser.ID,
			UserId:    user.ID,
			ClusterId: majorCluster.ID,
			Role:      clusterRole,
		})
		if err != nil {
			return nil, errors.Wrap(err, "create cluster member")
		}
	}

	// cache allowed api list
	allowedApiList, err := t.GetUserApiPermissions(t.GinContext, accessToken.AccessToken, userToken.UserToken)
	if err != nil {
		return nil, err
	}
	userApiListCache := []cache.ApiRecord{}
	for _, item := range *allowedApiList {
		tmp := cache.ApiRecord{
			SecurityLevel: uint8(item.SecurityLevel),
			ApiMethod:     item.ApiMethod,
			ApiName:       item.ApiName,
			ApiRouter:     item.ApiRouter,
			ApiCode:       item.ApiCode,
			MenuParent:    item.MenuParent,
			ApiDomain:     item.ApiDomain,
			MenuType:      uint8(item.MenuType),
			ClientId:      item.ClientId,
		}
		userApiListCache = append(userApiListCache, tmp)
	}
	cache.NewSingleCache().Cache.GetOrSet(userToken.UserToken, cache.UserCacheData{
		ApiRecords: userApiListCache,
		UserName:   userInfo.PrUserName,
	})
	// ------------- this solution depreciated
	// todo deal with the case of admin user. how to set the organization;
	// isSaas as false, it will setup default admin/org/cluster; connect the idp admin to the defaul admin by
	// updating the Username and add email address

	allowedMenus, err := t.GetUserMenuPermissions(t.GinContext,accessToken.AccessToken,userToken.UserToken)
	if err != nil {
		return nil, err
	}
	res := &thirdpartyauth.UserInfoAndMenus{
		UserDetails: *userInfo,
		MenuList: allowedMenus,
		UserToken: userToken.UserToken,
	}
	
	return res , nil

}

func (t thirdPartyAuthService) GetUserApiPermissionsSvc() (interface{}, error) {
	//todo db logic
	//uesrToken := "4fe1d1696b96444c886b64b26c1efc8d"
	allowedApiList, err := t.GetUserApiPermissions(t.GinContext, "", "")
	if err != nil {
		return nil, err
	}
	userApiListCache := []cache.ApiRecord{}
	for _, item := range *allowedApiList {
		tmp := cache.ApiRecord{
			SecurityLevel: uint8(item.SecurityLevel),
			ApiMethod:     item.ApiMethod,
			ApiName:       item.ApiName,
			ApiRouter:     item.ApiRouter,
			ApiCode:       item.ApiCode,
			MenuParent:    item.MenuParent,
			ApiDomain:     item.ApiDomain,
			MenuType:      uint8(item.MenuType),
			ClientId:      item.ClientId,
		}
		userApiListCache = append(userApiListCache, tmp)
	}
	//cache.NewSingleCache().Cache.GetOrSet(uesrToken,userApiListCache)

	return "success", nil
}

func (t thirdPartyAuthService) RefreshAccessToken() (*thirdpartyauth.TokenContainer, error) {
	//todo db logic
	refreshToken := "akjshdfkajshdf"
	return t.RefreshAssesToken(t.GinContext, refreshToken)
}

func returnUrl() string {
	return fmt.Sprintf("%s%s/auth", "l.AppCtx.APIEndpoint()", "l.AppCtx.WhoAmI()")
}
