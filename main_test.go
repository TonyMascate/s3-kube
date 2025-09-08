package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

func cleanupBucket(ctx context.Context, bucket string) error {
	// Vérifier si le bucket existe
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Lister et supprimer tous les objets
	opts := minio.ListObjectsOptions{
		Recursive: true,
	}
	for obj := range minioClient.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return obj.Err
		}
		err = minioClient.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return err
		}
	}

	// Supprimer le bucket
	return minioClient.RemoveBucket(ctx, bucket)
}

func TestS3Process(t *testing.T) {
	ctx := context.Background()

	// ⚠️ Initialiser minioClient AVANT tout cleanup
	var err error
	minioClient, err = initMinioWithCreds("minioadmin", "miniopassword")
	if err != nil {
		t.Fatalf("Erreur init MinIO: %v", err)
	}

	// Gin en mode test
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	setupRoutes(router)

	// -------------------------
	// Nettoyage complet avant tests
	// -------------------------
	cleanupBucket(ctx, "bucket-a")
	cleanupBucket(ctx, "bucket-b")

	// -------------------------
	// Partie 1 : bucket/fichier éphémères
	// -------------------------
	bucketA := "bucket-a"
	objectA := "file-a.txt"
	contentA := "Hello world from test A"

	// Créer bucket A
	req1, _ := http.NewRequest(http.MethodPut, "/"+bucketA, nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Lister buckets
	req2, _ := http.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), bucketA)

	// Uploader fichier A
	req3, _ := http.NewRequest(http.MethodPut, "/"+bucketA+"/"+objectA, bytes.NewBuffer([]byte(contentA)))
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	// Télécharger fichier A
	req4, _ := http.NewRequest(http.MethodGet, "/"+bucketA+"/"+objectA, nil)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
	body, _ := io.ReadAll(w4.Body)
	assert.Equal(t, contentA, string(body))

	// Supprimer fichier A
	req5, _ := http.NewRequest(http.MethodDelete, "/"+bucketA+"/"+objectA, nil)
	w5 := httptest.NewRecorder()
	router.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusNoContent, w5.Code)

	// Supprimer bucket A
	req6, _ := http.NewRequest(http.MethodDelete, "/"+bucketA, nil)
	w6 := httptest.NewRecorder()
	router.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusNoContent, w6.Code)

	// -------------------------
	// Partie 2 : bucket/fichier persistants
	// -------------------------
	bucketB := "bucket-b"
	objectB := "file-b.txt"
	contentB := "Hello world from test B"

	// Créer bucket B
	req7, _ := http.NewRequest(http.MethodPut, "/"+bucketB, nil)
	w7 := httptest.NewRecorder()
	router.ServeHTTP(w7, req7)
	assert.Equal(t, http.StatusOK, w7.Code)

	// Lister buckets
	req8, _ := http.NewRequest(http.MethodGet, "/", nil)
	w8 := httptest.NewRecorder()
	router.ServeHTTP(w8, req8)
	assert.Equal(t, http.StatusOK, w8.Code)
	assert.Contains(t, w8.Body.String(), bucketB)

	// Uploader fichier B
	req9, _ := http.NewRequest(http.MethodPut, "/"+bucketB+"/"+objectB, bytes.NewBuffer([]byte(contentB)))
	w9 := httptest.NewRecorder()
	router.ServeHTTP(w9, req9)
	assert.Equal(t, http.StatusOK, w9.Code)

	// Télécharger fichier B
	req10, _ := http.NewRequest(http.MethodGet, "/"+bucketB+"/"+objectB, nil)
	w10 := httptest.NewRecorder()
	router.ServeHTTP(w10, req10)
	assert.Equal(t, http.StatusOK, w10.Code)
	bodyB, _ := io.ReadAll(w10.Body)
	assert.Equal(t, contentB, string(bodyB))

	// ⚠ Ne PAS supprimer bucket B ni fichier B à la fin

}
