package main

import (
	"encoding/json"
	"errors"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	cas20200407 "github.com/alibabacloud-go/cas-20200407/v3/client"
	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/tjfoc/gmsm/x509"
	"github.com/yankeguo/rg"
)

type Options struct {
	CrtFile string `json:"crt_file" default:"/tls/tls.crt" validate:"required,file"`
	KeyFile string `json:"key_file" default:"/tls/tls.key" validate:"required,file"`
	Aliyun  *struct {
		AccessKeyId     string   `json:"access_key_id" validate:"required"`
		AccessKeySecret string   `json:"access_key_secret" validate:"required"`
		RegionId        string   `json:"region_id" validate:"required"`
		CDNDomains      []string `json:"cdn_domains"`
	} `json:"aliyun"`
}

func createAliyunCasClient(regionId, accessKeyId, accessKeySecret string) (*cas20200407.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	}
	config.Endpoint = tea.String("cas.aliyuncs.com")
	return cas20200407.NewClient(config)
}

func createAliyunCdnClient(regionId, accessKeyId, accessKeySecret string) (*cdn20180510.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	}
	config.Endpoint = tea.String("cdn.aliyuncs.com")
	return cdn20180510.NewClient(config)
}

func nameFroCertificate(crt *x509.Certificate) string {
	names := append([]string{crt.Subject.CommonName}, crt.DNSNames...)
	var shortest string
	for _, name := range names {
		if shortest == "" || len(name) < len(shortest) {
			shortest = name
		}
	}
	prefixes := make(map[string]struct{})
	for _, name := range names {
		if strings.HasSuffix(name, shortest) {
			name = strings.TrimSuffix(name, shortest)
		}
		if strings.HasPrefix(name, "*.") {
			name = strings.TrimPrefix(name, "*.")
		}
		name = strings.ReplaceAll(name, ".", "-")
		prefixes[name] = struct{}{}
	}
	return strings.Join(slices.Collect(maps.Keys(prefixes)), "-") + "-" + shortest
}

func main() {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Printf("exit with error: %s", err.Error())
		os.Exit(1)
	}()
	defer rg.Guard(&err)

	var opts Options
	buf := rg.Must(os.ReadFile("config.json"))
	rg.Must0(json.Unmarshal(buf, &opts))
	rg.Must0(defaults.Set(&opts))
	rg.Must0(validator.New().Struct(opts))

	crtPEM := rg.Must(os.ReadFile(opts.CrtFile))
	keyPEM := rg.Must(os.ReadFile(opts.KeyFile))

	crt := rg.Must(x509.ReadCertificateFromPem(crtPEM))

	if time.Now().After(crt.NotAfter) {
		err = errors.New("cert is expired")
		return
	}

	if opts.Aliyun != nil {
		casClient := rg.Must(createAliyunCasClient(opts.Aliyun.RegionId, opts.Aliyun.AccessKeyId, opts.Aliyun.AccessKeySecret))
		cdnClient := rg.Must(createAliyunCdnClient(opts.Aliyun.RegionId, opts.Aliyun.AccessKeyId, opts.Aliyun.AccessKeySecret))

		var existingCrtList []*cas20200407.ListUserCertificateOrderResponseBodyCertificateOrderList
		{
			req := &cas20200407.ListUserCertificateOrderRequest{
				OrderType: tea.String("UPLOAD"),
			}
			res := rg.Must(casClient.ListUserCertificateOrder(req))

			existingCrtList = res.Body.CertificateOrderList
		}

		var crtId int64

		for _, existingCrt := range existingCrtList {
			if existingCrt.SerialNo != nil && *existingCrt.SerialNo == crt.SerialNumber.String() {
				crtId = *existingCrt.CertificateId
				log.Printf("certificate %d already exists, skip uploading", crtId)
				goto crtFound
			}
		}

		{
			res := rg.Must(casClient.UploadUserCertificate(&cas20200407.UploadUserCertificateRequest{
				Name: tea.String(nameFroCertificate(crt)),
				Cert: tea.String(string(crtPEM)),
				Key:  tea.String(string(keyPEM)),
			}))

			crtId = *res.Body.CertId
			log.Printf("certificate %d uploaded, with dns names: %s", crtId, strings.Join(append([]string{crt.Subject.CommonName}, crt.DNSNames...), ", "))
		}

	crtFound:
		for _, cdnDomain := range opts.Aliyun.CDNDomains {
			var match bool
			{
				res := rg.Must(cdnClient.DescribeDomainCertificateInfo(&cdn20180510.DescribeDomainCertificateInfoRequest{
					DomainName: tea.String(cdnDomain),
				}))
				if res.Body.CertInfos != nil {
					for _, certInfo := range res.Body.CertInfos.CertInfo {
						if certInfo.CertId != nil && *certInfo.CertId == strconv.FormatInt(crtId, 10) {
							match = true
							log.Printf("certificate %d already bound to domain %s, skip binding", crtId, cdnDomain)
							break
						}
					}
				}
			}

			if !match {
				rg.Must(cdnClient.SetCdnDomainSSLCertificate(&cdn20180510.SetCdnDomainSSLCertificateRequest{
					DomainName: tea.String(cdnDomain),
					CertType:   tea.String("cas"),
					CertId:     tea.Int64(crtId),
				}))
				log.Printf("certificate %d bound to domain %s", crtId, cdnDomain)
			}
		}
	}

	if opts.Aliyun == nil {
		err = errors.New("aliyun is not configured")
	}
}
