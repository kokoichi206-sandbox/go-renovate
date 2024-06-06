package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func cfClient() *s3.Client {
	accountID := os.Getenv("ACCOUNT_ID")
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			HostnameImmutable: true,
			SigningRegion:     region,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("auto"),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		log.Fatal(err)
	}

	// **aws の** s3 client を作成する。
	client := s3.NewFromConfig(cfg)

	return client
}

func cfPreSignedURL() string {
	client := cfClient()

	pc := s3.NewPresignClient(
		client,
		// 10 秒で期限切れになる署名付き URL とする。
		// default は 15 分 (900 秒)。
		s3.WithPresignExpires(10*time.Second),
	)

	req, err := pc.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String("aaaaa"),
		Key:    aws.String("xxxx"),
	})

	if err != nil {
		return ""
	}

	fmt.Printf("req: %v\n", req)
	return req.URL
}

// https://docs.aws.amazon.com/AmazonS3/latest/userguide/example_s3_Scenario_PresignedUrl_section.html
func preSignedURL(ctx context.Context, objectName string) string {
	bucket := os.Getenv("S3_BUCKET")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg)

	pc := s3.NewPresignClient(
		client,
		// 10 秒で期限切れになる署名付き URL とする。
		// default は 15 分 (900 秒)。
		s3.WithPresignExpires(10*time.Second),
	)

	req, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	})

	if err != nil {
		return ""
	}

	fmt.Printf("req: %v\n", req)
	return req.URL
}

func upload(ctx context.Context, objectName string, object io.Reader, len int64) {
	client := cfClient()

	if _, err := client.PutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket:        aws.String("xxxxx"),
			Key:           aws.String("yyyy"),
			Body:          object,
			ContentLength: &len,
		},
		s3.WithAPIOptions(
			v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware,
		),
	); err != nil {
		log.Fatal(err)
	}
}

var driveURLRegex = regexp.MustCompile("https://drive.google.com/file/d/([^/]*)/view")

func fetchAndUpload(ctx context.Context, viewURL string, fileName string) {

	matched := driveURLRegex.FindStringSubmatch(viewURL)
	fmt.Printf("matched: %v\n", matched)
	if len(matched) < 2 {
		panic("No match found")
	}

	downloadURL := ""
	fmt.Printf("downloadURL: %v\n", downloadURL)

	res, err := http.Get(downloadURL)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	upload(ctx, fileName, res.Body, res.ContentLength)
}

func main() {
	ctx := context.Background()
	stats, _ := os.Stat("data.zip")

	f, _ := os.Open("data.zip")
	upload(ctx, "nanka.pdf", f, stats.Size())
	fmt.Printf("cfPreSignedURL(): %v\n", cfPreSignedURL())
	return
	// ctx := context.Background()
	preSignedURL(ctx, "nanka.pdf")
	return
	fetchAndUpload(ctx, os.Getenv("DRIVE_URL"), "nanka.pdf")
}
