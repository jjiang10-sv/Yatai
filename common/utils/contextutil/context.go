package contextutil

import (
	"context"
	"net/http"

	"github.com/bentoml/yatai/common/utils/logx"
	"github.com/bentoml/yatai/common/utils/xidx"
)

// Context helpers
type ServiceContexts uint

// comes from headers
const RequestIDCtxID ServiceContexts = 0
const UserAgentCtxID ServiceContexts = 1
const xxxAuthTokenCtxID ServiceContexts = 2

// comes from oauth access token
const SSOIDCtxID ServiceContexts = 3

// comes from client cert
const CertSubjectCtxID ServiceContexts = 4
const CertCNCtxID ServiceContexts = 5

// comes from query params
const DinsCtxID ServiceContexts = 6
const SiteIDCtxID ServiceContexts = 7
const AssetSiteIDCtxID ServiceContexts = 8
const TimezoneCtxID ServiceContexts = 9
const MyxxxUserIDCtxID ServiceContexts = 10
const VinCtxID ServiceContexts = 11
const LanguageCtxID ServiceContexts = 12
const CurrencyCodeCtxID ServiceContexts = 13
const VinsCtxId ServiceContexts = 14

// comes from request body
const RequestBodyCtxID ServiceContexts = 15

// NewServiceContextWithValue returns a copy of the passed in context
// with the key:value added
func NewServiceContextWithValue(ctx context.Context, key ServiceContexts, val string) context.Context {
	return context.WithValue(ctx, key, val)
}

// type ReqContext struct {
// 	context.Context
// 	xid string
// 	logx.ReqLogger
// }

// func NewReqContext() ReqContext {
// 	rc := ReqContext{
// 		Context: context.Background(),
// 		xid:     xidx.GenXid(),
// 	}
// 	rc.ReqLogger = logx.MustNewReqLogger(rc.xid)
// 	return rc
// }

func NewReqLogger() logx.ReqLogger {

	return logx.MustNewReqLogger(xidx.GenXid())

}

// NewServiceContextWithGenericValue returns a copy of the passed in context
// with the key:generic value added
func NewServiceContextWithGenericValue(ctx context.Context, key ServiceContexts, val interface{}) context.Context {
	return context.WithValue(ctx, key, val)
}

// GetStringValueFromContext returns value from context by given key
// assumes all values will be a string
func GetStringValueFromContext(ctx context.Context, key ServiceContexts) string {
	val, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return val
}

// GetGenericValueFromContext returns value from context by given key
// no type assertion is done on the value
func GetGenericValueFromContext(ctx context.Context, key ServiceContexts) interface{} {
	return ctx.Value(key)
}

func GetRequestIdFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, RequestIDCtxID)
}

func GetUserAgentFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, UserAgentCtxID)
}

func GetxxxAuthTokenFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, xxxAuthTokenCtxID)
}

func GetSSOIDFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, SSOIDCtxID)
}

func GetCertSubjectFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, CertSubjectCtxID)
}

func GetCertCNFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, CertCNCtxID)
}

func GetDinsFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, DinsCtxID)
}

func GetSiteIDFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, SiteIDCtxID)
}

func GetAssetSiteIDFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, AssetSiteIDCtxID)
}

func GetTimezoneFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, TimezoneCtxID)
}

func GetMyxxxUserIDFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, MyxxxUserIDCtxID)
}

func GetVinFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, VinCtxID)
}

func GetVinsFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, VinsCtxId)
}

func GetLanguageFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, LanguageCtxID)
}

func GetCurrencyCodeFromContext(ctx context.Context) string {
	return GetStringValueFromContext(ctx, CurrencyCodeCtxID)
}

func GetRequestBodyFromContext(ctx context.Context) interface{} {
	return GetGenericValueFromContext(ctx, RequestBodyCtxID)
}

// ServiceHeadersFromContext adds any relevant values in context to request headers
func ServiceHeadersFromContext(ctx context.Context, headers http.Header) http.Header {
	headers = headers.Clone()
	headers.Set("x-txid", GetRequestIdFromContext(ctx))
	return headers
}
