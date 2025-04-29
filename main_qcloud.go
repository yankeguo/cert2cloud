package main

import "github.com/yankeguo/rg"

func updateQcloud(certOpts *CertOptions, qcloudOpts *QcloudOptions) (err error) {
	defer rg.Guard(&err)

	localCert, localCertName := rg.Must2(certOpts.CreateCertificate())

	client := rg.Must(qcloudOpts.CreateSslClient())

	_ = localCert
	_ = localCertName
	_ = client

	return
}
