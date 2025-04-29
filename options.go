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

	"github.com/yankeguo/rg"
)

type CertOptions struct {
	NamePrefix  string `json:"name_prefix"`
	CertPEM     string `json:"cert_pem"`
	CertPEMFile string `json:"cert_pem_file"`
	KeyPEM      string `json:"key_pem"`
	KeyPEMFile  string `json:"key_pem_file"`
}

func (opts *CertOptions) Validate() {
	requireField("cert.name_prefix", &opts.NamePrefix)
	requireFieldWithFile("cert.cert_pem", &opts.CertPEM, opts.CertPEMFile)
	requireFieldWithFile("cert.key_pem", &opts.KeyPEM, opts.KeyPEMFile)
}

func (opts *CertOptions) CreateCertificate() (cert *x509.Certificate, name string, err error) {
	defer rg.Guard(&err)
	block, _ := pem.Decode([]byte(opts.CertPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		err = errors.New("invalid cert.cert_pem")
		return
	}

	cert = rg.Must(x509.ParseCertificate(block.Bytes))

	if cert.SerialNumber == nil {
		err = errors.New("cert.cert_pem serial number is nil")
		return
	}

	if time.Now().After(cert.NotAfter) {
		err = errors.New("cert.cert_pem is expired")
		return
	}

	name = opts.NamePrefix + "-" + cert.NotAfter.Format("20060102150405")
	return
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

	opts.Cert.Validate()

	if opts.Aliyun != nil {
		opts.Aliyun.Validate()
	}

	if opts.Qcloud != nil {
		opts.Qcloud.Validate()
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
