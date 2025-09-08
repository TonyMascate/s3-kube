package main

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func initMinioWithCreds(accessKey, secretKey string) (*minio.Client, error) {
	endpoint := getenv("MINIO_ENDPOINT", "localhost:9000")

	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
}
