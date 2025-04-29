package main

import (
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
)

type QcloudOptions struct {
	SecretID      string   `json:"secret_id"`
	SecretIDFile  string   `json:"secret_id_file"`
	SecretKey     string   `json:"secret_key"`
	SecretKeyFile string   `json:"secret_key_file"`
	ResourceTypes []string `json:"resource_types"`
}

func (opts *QcloudOptions) Validate() {
	requireFieldWithFile("qcloud.secret_id", &opts.SecretID, opts.SecretIDFile)
	requireFieldWithFile("qcloud.secret_key", &opts.SecretKey, opts.SecretKeyFile)
}

func (opts *QcloudOptions) CreateSslClient() (*ssl.Client, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ssl.tencentcloudapi.com"

	return ssl.NewClient(
		common.NewCredential(
			opts.SecretID,
			opts.SecretKey,
		),
		"",
		cpf,
	)
}
