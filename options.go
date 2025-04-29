package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	cas20200407 "github.com/alibabacloud-go/cas-20200407/v3/client"
	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/yankeguo/rg"
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

type QcloudOptions struct {
	SecretID      string `json:"secret_id"`
	SecretIDFile  string `json:"secret_id_file"`
	SecretKey     string `json:"secret_key"`
	SecretKeyFile string `json:"secret_key_file"`
}

type CertOptions struct {
	NamePrefix  string            `json:"name_prefix"`
	CertPEM     string            `json:"cert_pem"`
	CertPEMFile string            `json:"cert_pem_file"`
	KeyPEM      string            `json:"key_pem"`
	KeyPEMFile  string            `json:"key_pem_file"`
	Cert        *x509.Certificate `json:"-"`
	Name        string            `json:"-"`
}

type Options struct {
	Cert   *CertOptions   `json:"cert"`
	Aliyun *AliyunOptions `json:"aliyun"`
	Qcloud *QcloudOptions `json:"qcloud"`
}

func LoadOptions(file string) (opts Options, err error) {
	defer rg.Guard(&err)

	buf := rg.Must(os.ReadFile(file))
	rg.Must0(json.Unmarshal(buf, &opts))

	if opts.Cert == nil {
		err = errors.New("missing cert options")
		return
	}

	requireField("cert.name_prefix", &opts.Cert.NamePrefix)
	requireFieldWithFile("cert.cert_pem", &opts.Cert.CertPEM, opts.Cert.CertPEMFile)
	requireFieldWithFile("cert.key_pem", &opts.Cert.KeyPEM, opts.Cert.KeyPEMFile)

	block, _ := pem.Decode([]byte(opts.Cert.CertPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		err = errors.New("invalid cert.cert_pem")
		return
	}

	opts.Cert.Cert = rg.Must(x509.ParseCertificate(block.Bytes))

	if opts.Cert.Cert.SerialNumber == nil {
		err = errors.New("cert.cert_pem serial number is nil")
		return
	}

	if time.Now().After(opts.Cert.Cert.NotAfter) {
		err = errors.New("cert.cert_pem is expired")
		return
	}

	opts.Cert.Name = opts.Cert.NamePrefix + "-" + opts.Cert.Cert.NotAfter.Format("20060102150405")

	if opts.Aliyun != nil {
		requireFieldWithFile("aliyun.access_key_id", &opts.Aliyun.AccessKeyID, opts.Aliyun.AccessKeyIDFile)
		requireFieldWithFile("aliyun.access_key_secret", &opts.Aliyun.AccessKeySecret, opts.Aliyun.AccessKeySecretFile)
		requireFieldWithFile("aliyun.region_id", &opts.Aliyun.RegionID, opts.Aliyun.RegionIDFile)
	}

	if opts.Qcloud != nil {
		requireFieldWithFile("qcloud.secret_id", &opts.Qcloud.SecretID, opts.Qcloud.SecretIDFile)
		requireFieldWithFile("qcloud.secret_key", &opts.Qcloud.SecretKey, opts.Qcloud.SecretKeyFile)
	}

	return
}

func requireField[T comparable](name string, field *T) {
	var empty T

	if *field != empty {
		return
	}

	panic(fmt.Errorf("missing field %s", name))
}

func requireFieldWithFile(name string, field *string, file string) (err error) {
	if *field != "" {
		return
	}

	if file == "" {
		panic(fmt.Errorf("missing field %s and %s_file", name, name))
	}

	buf := bytes.TrimSpace(rg.Must(os.ReadFile(file)))

	if len(buf) == 0 {
		panic(fmt.Errorf("field %s_file refers an empty file: %s", name, file))
	}

	*field = string(buf)

	return
}
