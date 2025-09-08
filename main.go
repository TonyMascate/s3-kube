package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

var minioClient *minio.Client

func main() {

	var err error
	minioClient, err = initMinioWithCreds("minioadmin", "miniopassword")
	if err != nil {
		log.Fatalf("Erreur init MinIO: %v", err)
	}

	r := gin.Default()

	// Routes principales (compatibles S3)
	setupRoutes(r)

	log.Println("API S3-like démarrée sur :8080")
	r.Run(":8080")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

//
// --------- ROUTES S3 ---------
//

// ✅ CreateBucket
func createBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	ctx := context.Background()

	err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, _ := minioClient.BucketExists(ctx, bucket)
		if exists {
			c.XML(http.StatusConflict, gin.H{
				"Error": gin.H{
					"Code":       "BucketAlreadyOwnedByYou",
					"Message":    "Your previous request to create the named bucket succeeded and you already own it.",
					"BucketName": bucket,
				},
			})
			return
		}
		c.XML(http.StatusInternalServerError, gin.H{
			"Error": gin.H{
				"Code":    "InternalError",
				"Message": err.Error(),
			},
		})
		return
	}
	c.Status(http.StatusOK)
}

// ✅ DeleteBucket
func deleteBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	ctx := context.Background()

	err := minioClient.RemoveBucket(ctx, bucket)
	if err != nil {
		exists, _ := minioClient.BucketExists(ctx, bucket)
		if !exists {
			c.XML(http.StatusNotFound, gin.H{
				"Error": gin.H{
					"Code":       "NoSuchBucket",
					"Message":    "The specified bucket does not exist",
					"BucketName": bucket,
				},
			})
			return
		}
		c.XML(http.StatusConflict, gin.H{
			"Error": gin.H{
				"Code":       "BucketNotEmpty",
				"Message":    "The bucket you tried to delete is not empty",
				"BucketName": bucket,
			},
		})
		return
	}
	c.Status(http.StatusNoContent)
}

// ✅ ListBuckets
func listBuckets(c *gin.Context) {
	ctx := context.Background()
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		c.XML(http.StatusInternalServerError, gin.H{
			"Error": gin.H{
				"Code":    "InternalError",
				"Message": err.Error(),
			},
		})
		return
	}

	type Bucket struct {
		Name         string    `xml:"Name"`
		CreationDate time.Time `xml:"CreationDate"`
	}
	var bucketList []Bucket
	for _, b := range buckets {
		bucketList = append(bucketList, Bucket{
			Name:         b.Name,
			CreationDate: b.CreationDate,
		})
	}

	c.XML(http.StatusOK, gin.H{
		"ListAllMyBucketsResult": gin.H{
			"xmlns": "http://s3.amazonaws.com/doc/2006-03-01/",
			"Buckets": gin.H{
				"Bucket": bucketList,
			},
			"Owner": gin.H{
				"ID":          "local-minio",
				"DisplayName": "local-user",
			},
		},
	})
}

// ✅ PutObject (upload)
func uploadFile(c *gin.Context) {
	bucket := c.Param("bucket")
	object := c.Param("object")
	ctx := context.Background()

	// Vérifier existence du bucket
	exists, _ := minioClient.BucketExists(ctx, bucket)
	if !exists {
		c.XML(http.StatusNotFound, gin.H{
			"Error": gin.H{
				"Code":       "NoSuchBucket",
				"Message":    "The specified bucket does not exist",
				"BucketName": bucket,
			},
		})
		return
	}

	// Lire body
	data, err := c.GetRawData()
	if err != nil {
		c.XML(http.StatusBadRequest, gin.H{
			"Error": gin.H{
				"Code":    "InvalidRequest",
				"Message": err.Error(),
			},
		})
		return
	}

	// Uploader vers MinIO
	_, err = minioClient.PutObject(ctx, bucket, object,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{})
	if err != nil {
		c.XML(http.StatusInternalServerError, gin.H{
			"Error": gin.H{
				"Code":    "InternalError",
				"Message": err.Error(),
			},
		})
		return
	}

	c.Status(http.StatusOK)
}

// ✅ GetObject (download)
func downloadFile(c *gin.Context) {
	bucket := c.Param("bucket")
	object := c.Param("object")
	ctx := context.Background()

	obj, err := minioClient.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		c.XML(http.StatusNotFound, gin.H{
			"Error": gin.H{
				"Code":    "NoSuchKey",
				"Message": "The specified key does not exist.",
			},
		})
		return
	}
	stat, err := obj.Stat()
	if err != nil {
		c.XML(http.StatusNotFound, gin.H{
			"Error": gin.H{
				"Code":    "NoSuchKey",
				"Message": "The specified key does not exist.",
			},
		})
		return
	}

	// conversion correcte du int64 en string
	c.Header("Content-Length", strconv.FormatInt(stat.Size, 10))
	c.Header("Content-Type", stat.ContentType)
	http.ServeContent(c.Writer, c.Request, object, stat.LastModified, obj)
}

// ✅ DeleteObject
func deleteFile(c *gin.Context) {
	bucket := c.Param("bucket")
	object := c.Param("object")
	ctx := context.Background()

	err := minioClient.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
	if err != nil {
		c.XML(http.StatusInternalServerError, gin.H{
			"Error": gin.H{
				"Code":    "InternalError",
				"Message": err.Error(),
			},
		})
		return
	}
	c.Status(http.StatusNoContent)
}
