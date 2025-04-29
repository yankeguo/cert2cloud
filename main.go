package main

import (
	"errors"
	"flag"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	cas20200407 "github.com/alibabacloud-go/cas-20200407/v3/client"
	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v6/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/yankeguo/rg"
)

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
		optConf string
	)

	flag.StringVar(&optConf, "conf", "config.json", "config file")
	flag.Parse()

	log.Println("loading config from:", optConf)

	opts := rg.Must(LoadOptions(optConf))

	if opts.Aliyun != nil {
		casClient := rg.Must(opts.Aliyun.CreateCasClient())
		cdnClient := rg.Must(opts.Aliyun.CreateCdnClient())

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

			if existingSerial.Cmp(opts.Cert.Cert.SerialNumber) == 0 {
				crtId = *existingCrt.CertificateId
				crtName = *existingCrt.Name
				log.Printf("certificate %d already exists, skip uploading", crtId)
				goto crtFound
			}
		}

		{
			crtName = opts.Cert.Name
			res := rg.Must(casClient.UploadUserCertificate(&cas20200407.UploadUserCertificateRequest{
				Name: tea.String(crtName),
				Cert: tea.String(opts.Cert.CertPEM),
				Key:  tea.String(opts.Cert.KeyPEM),
			}))

			crtId = *res.Body.CertId
			log.Printf("certificate %d uploaded, with dns names: %s", crtId, strings.Join(append([]string{opts.Cert.Cert.Subject.CommonName}, opts.Cert.Cert.DNSNames...), ", "))
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
