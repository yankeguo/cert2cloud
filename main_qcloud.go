package main

import (
	"log"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
	"github.com/yankeguo/rg"
)

func updateQcloud(certOpts *CertOptions, qcloudOpts *QcloudOptions) (err error) {
	defer rg.Guard(&err)

	localCert, localCertName := rg.Must2(certOpts.CreateCertificate())

	localDomainMark := cleanJoined(localCert.DNSNames)

	client := rg.Must(qcloudOpts.CreateSslClient())

	var (
		cloudCertID          string
		expiringCloudCertIDs []string
	)

	{
		req := ssl.NewDescribeCertificatesRequest()
		req.Limit = common.Uint64Ptr(1000)
		req.CertificateType = common.StringPtr("SVR")
		req.ExpirationSort = common.StringPtr("ASC")
		req.Upload = common.Int64Ptr(1)

		res := rg.Must(client.DescribeCertificates(req))

		for _, item := range res.Response.Certificates {
			var (
				end  time.Time
				err1 error
			)
			if end, err1 = time.Parse(time.DateTime, *item.CertEndTime); err1 != nil {
				// ignore invalid date
				continue
			}
			if localDomainMark == cleanJoinedPtr(item.SubjectAltName) &&
				timeDiff(end, localCert.NotAfter) < time.Hour*48 {
				cloudCertID = *item.CertificateId
				log.Println("Found existing certificate:", localCertName, "ID:", cloudCertID)
				break
			}
		}

		for _, item := range res.Response.Certificates {
			if cloudCertID != "" && cloudCertID == *item.CertificateId {
				// ignoring certificate it-self
				continue
			}
			if localDomainMark != cleanJoinedPtr(item.SubjectAltName) {
				// ignore mismatched domains
				continue
			}
			var (
				end  time.Time
				err1 error
			)
			if end, err1 = time.Parse(time.DateTime, *item.CertEndTime); err1 != nil {
				// ignore invalid date
				continue
			}
			if end.After(localCert.NotAfter) {
				// ignore newer certificate
				continue
			}
			expiringCloudCertIDs = append(expiringCloudCertIDs, *item.CertificateId)
		}

		log.Println("Found", len(expiringCloudCertIDs), "expiring certificates for domain:", localDomainMark)
	}

	if cloudCertID == "" {
		req := ssl.NewUploadCertificateRequest()
		req.CertificatePublicKey = common.StringPtr(certOpts.CertPEM)
		req.CertificatePrivateKey = common.StringPtr(certOpts.KeyPEM)
		req.CertificateType = common.StringPtr("SVR")
		req.Alias = common.StringPtr(localCertName)
		req.Repeatable = common.BoolPtr(false)

		res := rg.Must(client.UploadCertificate(req))

		cloudCertID = *res.Response.CertificateId
		log.Println("Uploaded new certificate:", localCertName, "ID:", cloudCertID)
	}

	for _, expiringCertID := range expiringCloudCertIDs {
		req := ssl.NewUpdateCertificateInstanceRequest()
		req.OldCertificateId = common.StringPtr(expiringCertID)
		req.ResourceTypes = common.StringPtrs(qcloudOpts.ResourceTypes)
		req.CertificateId = common.StringPtr(cloudCertID)

		for {
			res := rg.Must(client.UpdateCertificateInstance(req))

			if *res.Response.DeployRecordId > 0 {
				break
			}

			time.Sleep(time.Second * 5)
		}

		log.Println("Initialized replacing expiring certificate:", expiringCertID, "with new certificate ID:", cloudCertID)
	}

	return
}
