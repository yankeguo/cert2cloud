package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"math/big"
	"os"
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
	shortest = strings.TrimPrefix(shortest, "*.")
	shortest = strings.ReplaceAll(shortest, ".", "-")
	return shortest + "-" + crt.NotAfter.Format("20060102150405")
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

	var (
		optConfig string
	)

	flag.StringVar(&optConfig, "conf", "config.json", "config file")
	flag.Parse()
	log.Printf("loading config from %s", optConfig)

	var opts Options
	buf := rg.Must(os.ReadFile(optConfig))
	rg.Must0(json.Unmarshal(buf, &opts))
	rg.Must0(defaults.Set(&opts))
	rg.Must0(validator.New().Struct(opts))

	crtPEM := rg.Must(os.ReadFile(opts.CrtFile))
	keyPEM := rg.Must(os.ReadFile(opts.KeyFile))

	crt := rg.Must(x509.ReadCertificateFromPem(crtPEM))

	if crt.SerialNumber == nil {
		err = errors.New("cert serial number is nil")
		return
	}

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

		var (
			crtId   int64
			crtName string
		)

		for _, existingCrt := range existingCrtList {
			if existingCrt.SerialNo == nil || existingCrt.CertificateId == nil {
				continue
			}

			existingSerial := big.NewInt(0)
			existingSerial.SetString(*existingCrt.SerialNo, 16)

			if existingSerial.Cmp(crt.SerialNumber) == 0 {
				crtId = *existingCrt.CertificateId
				crtName = *existingCrt.Name
				log.Printf("certificate %d already exists, skip uploading", crtId)
				goto crtFound
			}
		}

		{
			crtName = nameFroCertificate(crt)
			res := rg.Must(casClient.UploadUserCertificate(&cas20200407.UploadUserCertificateRequest{
				Name: tea.String(crtName),
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
					DomainName:  tea.String(cdnDomain),
					CertType:    tea.String("cas"),
					CertId:      tea.Int64(crtId),
					CertName:    tea.String(crtName),
					SSLProtocol: tea.String("on"),
				}))
				log.Printf("certificate %d bound to domain %s", crtId, cdnDomain)
			}
		}
	}

	if opts.Aliyun == nil {
		err = errors.New("aliyun is not configured")
	}
}
