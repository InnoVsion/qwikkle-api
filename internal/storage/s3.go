package storage

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Presigner struct {
	presignClient *s3.PresignClient
}

type S3Config struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func NewS3Presigner(ctx context.Context, cfg S3Config) (*S3Presigner, error) {
	if cfg.Region == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, NotConfiguredError{}
	}

	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	}

	if cfg.Endpoint != "" {
		loadOptions = append(loadOptions, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == s3.ServiceID {
					return aws.Endpoint{
						URL:               cfg.Endpoint,
						HostnameImmutable: true,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.Endpoint != ""
	})

	return &S3Presigner{
		presignClient: s3.NewPresignClient(client),
	}, nil
}

func (p *S3Presigner) PresignPutObject(
	ctx context.Context,
	bucket string,
	key string,
	contentType string,
	contentLength int64,
	expiry time.Duration,
) (PresignResult, error) {
	if bucket == "" || key == "" {
		return PresignResult{}, errors.New("bucket and key are required")
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	out, err := p.presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return PresignResult{}, err
	}

	headers := headerToMap(out.SignedHeader)
	for k, v := range DefaultHeaders(contentType, contentLength) {
		headers[k] = v
	}

	return PresignResult{
		URL:     out.URL,
		Method:  "PUT",
		Headers: headers,
		Expires: time.Now().Add(expiry),
	}, nil
}

func (p *S3Presigner) PresignGetObject(ctx context.Context, bucket string, key string, expiry time.Duration) (PresignResult, error) {
	if bucket == "" || key == "" {
		return PresignResult{}, errors.New("bucket and key are required")
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	out, err := p.presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return PresignResult{}, err
	}

	return PresignResult{
		URL:     out.URL,
		Method:  "GET",
		Headers: headerToMap(out.SignedHeader),
		Expires: time.Now().Add(expiry),
	}, nil
}

func headerToMap(h map[string][]string) map[string]string {
	out := map[string]string{}
	for k, values := range h {
		if len(values) == 0 {
			continue
		}
		out[k] = values[0]
	}
	return out
}
