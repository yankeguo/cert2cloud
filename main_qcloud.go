package main

import (
	"log"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	qcloud_errors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
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
			endTime, err1 := time.Parse(time.DateTime, *item.CertEndTime)

			if err1 != nil {
				// ignore invalid date
				continue
			}

			if localDomainMark == cleanJoinedPtr(item.SubjectAltName) &&
				timeDiff(endTime, localCert.NotAfter) < time.Hour*48 {
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

			endTime, err1 := time.Parse(time.DateTime, *item.CertEndTime)

			if err1 != nil {
				// ignore invalid date
				continue
			}

			if endTime.After(localCert.NotAfter) {
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

		for product, regions := range qcloudOpts.ResourceRegions {
			req.ResourceTypesRegions = append(req.ResourceTypesRegions, &ssl.ResourceTypeRegions{
				ResourceType: common.StringPtr(product),
				Regions:      common.StringPtrs(regions),
			})
		}

		log.Println("Replacing expiring certificate:", expiringCertID, "with new certificate ID:", cloudCertID, req.ToJsonString())

		rg.Must0(safeQcloudUpdateCertificate(client, req))
	}

	return
}

func safeQcloudUpdateCertificate(client *ssl.Client, req *ssl.UpdateCertificateInstanceRequest) error {
	for {
		res, err := client.UpdateCertificateInstance(req)

		if err != nil {
			if qErr, ok := err.(*qcloud_errors.TencentCloudSDKError); ok {
				if qErr.Code == "FailedOperation.CertificateDeployInstanceEmpty" {
					log.Println("Ignored Certificate deploy instance empty error")
					return nil
				}
			}
			return err
		}

		if *res.Response.DeployRecordId > 0 {
			return nil
		}

		time.Sleep(time.Second * 2)
	}
}
