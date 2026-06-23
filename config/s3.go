package config

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var S3Client *s3.Client
var S3BucketName string

// ConnectS3 initializes the AWS SDK and S3 Client using environment variables.
// It automatically picks up AWS_REGION, AWS_ACCESS_KEY_ID, and AWS_SECRET_ACCESS_KEY.
func ConnectS3() {
	S3BucketName = getEnv("S3_BUCKET_NAME", "my-contact-uploads-qa")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load AWS SDK config, %v", err)
	}

	S3Client = s3.NewFromConfig(cfg)
	fmt.Printf("AWS S3 Client initialized successfully for bucket '%s'!\n", S3BucketName)
}
