package cache

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	//"github.com/jellydator/ttlcache/v3/cache"
)

//type APICache ttlcache.Cache[string, []ApiRecord]

type APICache struct {
	Cache *ttlcache.Cache[string, UserCacheData]
}

var (
	//instance *ttl.Cache[string, []ApiRecord]
	instance *APICache
	once     sync.Once
)

type UserCacheData struct {
	UserName   string
	ApiRecords []ApiRecord
}

// "securityLevel"：1，//安全级别
// "apiMethod"："GET"，//请求方式POSTGETPUT
// "apiName"："home"，//api名称
// "apiRouter"："/index"，//api路由
// "apiCode"："xxxxxx"，//api编码
// "menuParent"："0"，//父菜单编码
// "apiDomain"："http：//127.0.0.1"，//api域名
// "menuType"：9，//类型(9：接口)
// "client_id"："xxxxxx"//子平台clientid
type ApiRecord struct {
	SecurityLevel uint8  `json:"securityLevel"`
	ApiMethod     string `json:"apiMethod"`
	ApiName       string `json:"apiName"`
	ApiRouter     string `json:"apiRouter"`
	ApiCode       string `json:"apiCode"`
	MenuParent    string `json:"menuParent"`
	ApiDomain     string `json:"apiDomain"`
	MenuType      uint8  `json:"menuType"`
	ClientId      string `json:"client_id"`
}

func NewSingleCache() *APICache {
	once.Do(func() {
		// todo handle error
		cache := ttlcache.New[string, UserCacheData](
			ttlcache.WithTTL[string, UserCacheData](2 * time.Hour),
		)
		go cache.Start() // starts automatic expired item deletion
		instance = &APICache{Cache: cache}
	})
	return instance

}

func (t *APICache) IsUserTokenValid(userToken string) bool {
	return t.Cache.Has(userToken)

}

// /api/v1/auth/current
func (t *APICache) IsPermitted(userToken, apiPath, apiMethod string) bool {
	apiPermitted := t.Cache.Get(userToken)
	pattern := regexp.MustCompile(`{([^{}]+)}`)

	// Replace "{apiCode}" with "apiCode"
	apiPath = pattern.ReplaceAllString(apiPath, "$1")
	apiMethod = strings.ToLower(apiMethod)

	for _, item := range apiPermitted.Value().ApiRecords {
		if item.ApiMethod == apiMethod && item.ApiRouter == apiPath {
			return true
		}
	}
	return false
}

func (t *APICache) CheckAuth(userToken, apiPath, apiMethod string) bool {
	if t.IsUserTokenValid(userToken) {
		return t.IsPermitted(userToken, apiMethod, apiPath)
	} else {

	}
	apiPermitted := t.Cache.Get(userToken)
	for _, item := range apiPermitted.Value().ApiRecords {
		if item.ApiMethod == apiMethod && item.ApiRouter == apiPath {
			return true
		}
	}
	return false
}
