package models

import (
	"github.com/bentoml/yatai-schemas/modelschemas"
)

type AccessToken struct {
	BaseModel
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Type         string `json:"token_type"`
	ExpiredIn    uint16 `json:"expires_in"`
}

// db table name
const ResourceTypeAccessToken modelschemas.ResourceType = "yatai_third_party_auth"

func (a *AccessToken) GetResourceType() modelschemas.ResourceType {
	return ResourceTypeAccessToken
}

// func (a *AccessToken) IsExpired() bool {
// 	if a.ExpiredIn == nil {
// 		return false
// 	}
// 	return time.Now().After(*a.ExpiredIn)
// }
