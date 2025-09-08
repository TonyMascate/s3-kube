package main

import "github.com/gin-gonic/gin"

func setupRoutes(r *gin.Engine) {
	r.GET("/", listBuckets)
	r.PUT("/:bucket", createBucket)
	r.DELETE("/:bucket", deleteBucket)
	r.PUT("/:bucket/:object", uploadFile)
	r.GET("/:bucket/:object", downloadFile)
	r.DELETE("/:bucket/:object", deleteFile)
}
