package main

import (
	"log"
	"math/big"
	"strconv"
	"strings"

	cas20200407 "github.com/alibabacloud-go/cas-20200407/v3/client"
	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v6/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/yankeguo/rg"
)

func updateAliyun(certOpts *CertOptions, aliyunOpts *AliyunOptions) (err error) {
	defer rg.Guard(&err)

	casClient := rg.Must(aliyunOpts.CreateCasClient())
	cdnClient := rg.Must(aliyunOpts.CreateCdnClient())

	localCert, localCertName := rg.Must2(certOpts.CreateCertificate())

	var (
		cloudCertID   int64
		cloudCertName string
	)

	// search cloud certificate from list
	{
		var existingCrtList []*cas20200407.ListUserCertificateOrderResponseBodyCertificateOrderList
		{
			req := &cas20200407.ListUserCertificateOrderRequest{
				OrderType: tea.String("UPLOAD"),
			}
			res := rg.Must(casClient.ListUserCertificateOrder(req))

			existingCrtList = res.Body.CertificateOrderList
		}

		for _, existingCrt := range existingCrtList {
			if existingCrt.SerialNo == nil || existingCrt.CertificateId == nil {
				continue
			}

			existingSerial := big.NewInt(0)
			existingSerial.SetString(*existingCrt.SerialNo, 16)

			if existingSerial.Cmp(localCert.SerialNumber) == 0 {
				cloudCertID = *existingCrt.CertificateId
				cloudCertName = *existingCrt.Name
				log.Printf("certificate %d already exists, skip uploading", cloudCertID)
				break
			}
		}
	}

	if cloudCertID == 0 || cloudCertName == "" {
		// create certificate
		res := rg.Must(casClient.UploadUserCertificate(&cas20200407.UploadUserCertificateRequest{
			Name: tea.String(localCertName),
			Cert: tea.String(certOpts.CertPEM),
			Key:  tea.String(certOpts.KeyPEM),
		}))

		cloudCertID = *res.Body.CertId
		cloudCertName = localCertName
		log.Printf("certificate %d uploaded, with dns names: %s", cloudCertID, strings.Join(append([]string{localCert.Subject.CommonName}, localCert.DNSNames...), ", "))
	}

	// update domain certificate
cdnLoop:
	for _, cdnDomain := range aliyunOpts.CDNDomains {

		{
			res := rg.Must(cdnClient.DescribeDomainCertificateInfo(&cdn20180510.DescribeDomainCertificateInfoRequest{
				DomainName: tea.String(cdnDomain),
			}))
			if res.Body.CertInfos != nil {
				for _, certInfo := range res.Body.CertInfos.CertInfo {
					if certInfo.CertId != nil && *certInfo.CertId == strconv.FormatInt(cloudCertID, 10) {
						log.Printf("certificate %d already bound to domain %s, skip binding", cloudCertID, cdnDomain)
						continue cdnLoop
					}
				}
			}
		}

		rg.Must(cdnClient.SetCdnDomainSSLCertificate(&cdn20180510.SetCdnDomainSSLCertificateRequest{
			DomainName:  tea.String(cdnDomain),
			CertType:    tea.String("cas"),
			CertId:      tea.Int64(cloudCertID),
			CertName:    tea.String(cloudCertName),
			SSLProtocol: tea.String("on"),
		}))
		log.Printf("certificate %d bound to domain %s", cloudCertID, cdnDomain)
	}

	return
}
