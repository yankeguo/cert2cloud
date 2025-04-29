package main

import (
	cas20200407 "github.com/alibabacloud-go/cas-20200407/v3/client"
	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

type AliyunOptions struct {
	AccessKeyID         string   `json:"access_key_id"`
	AccessKeyIDFile     string   `json:"access_key_id_file"`
	AccessKeySecret     string   `json:"access_key_secret"`
	AccessKeySecretFile string   `json:"access_key_secret_file"`
	RegionID            string   `json:"region_id"`
	RegionIDFile        string   `json:"region_id_file"`
	CDNDomains          []string `json:"cdn_domains"`
}

func (opts *AliyunOptions) Validate() {
	requireFieldWithFile("aliyun.access_key_id", &opts.AccessKeyID, opts.AccessKeyIDFile)
	requireFieldWithFile("aliyun.access_key_secret", &opts.AccessKeySecret, opts.AccessKeySecretFile)
	requireFieldWithFile("aliyun.region_id", &opts.RegionID, opts.RegionIDFile)
}

func (opts *AliyunOptions) CreateCasClient() (*cas20200407.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(opts.AccessKeyID),
		AccessKeySecret: tea.String(opts.AccessKeySecret),
	}
	config.Endpoint = tea.String("cas.aliyuncs.com")
	return cas20200407.NewClient(config)
}

func (opts *AliyunOptions) CreateCdnClient() (*cdn20180510.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(opts.AccessKeyID),
		AccessKeySecret: tea.String(opts.AccessKeySecret),
	}
	config.Endpoint = tea.String("cdn.aliyuncs.com")
	return cdn20180510.NewClient(config)
}
