package yana

import (
	"context" // Why does this exist????
	"fmt"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"
)

const (
	FILENAME_MAX_LEN             = 255
	BUCKETNAME_MAX_LEN           = 63
	DEFAULT_BUCKET_SERVER_REGION = "us-east-1" // See https://min.io/docs/minio/linux/developers/go/API.html#MakeBucket:~:text=(defaults%20to%20us%2Deast%2D1).

)

// Bucketname == UUID of user
type Note struct {
	Owner User
	Name  string
}

type MinIOConfig struct {
	Url       string `yaml:"url"`
	AccessKey string `yaml:"accesskey"`
	SecretKey string `yaml:"secretkey"`
	UseSSL    bool   `yaml:"usessl"`
}

const MINIO_CONFIG_PATH = "db/minio.yml"

func readMinIOConfig(path string) (MinIOConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return MinIOConfig{}, err
	}

	config := MinIOConfig{}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return MinIOConfig{}, fmt.Errorf("Error in file %q: %w", path, err)
	}
	return config, err
}

// TODO: Maybe replace error with YanaError???
func generateMinIOClient() (*minio.Client, error) {
	minioConfig, err := readMinIOConfig(MINIO_CONFIG_PATH)
	if err != nil {
		return &minio.Client{}, fmt.Errorf("Error in yana.generateMinIOClient (Couldn't read minio config) -> err: %w")
	}
	minioOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKey, minioConfig.SecretKey, ""),
		Secure: minioConfig.UseSSL,
	}
	minioClient, err := minio.New(minioConfig.Url, minioOptions)
	if err != nil {
		return &minio.Client{}, fmt.Errorf("Error in yana.generateMinIOClient (Couldn't connect to minio) -> err: %w")
	}

	return minioClient, nil
}

func NewBucket(bucketName string) error {
	contextBackground := context.Background()
	minioClient, err := generateMinIOClient()
	if err != nil {
		return fmt.Errorf("yana.NewBucket() -> (Fail generating minioclient) Couldn't create bucket because: '%w'\n", err)
	}
	err = minioClient.MakeBucket(contextBackground, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("yana.NewBucket() -> Couldn't create bucket because: '%w'\n", err)
	}
	return nil
}

func NewNote(userId string, noteName string, content string) (minio.UploadInfo, error) {
	contextBackground := context.Background()
	minioClient, err := generateMinIOClient()
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewBucket() -> (Fail generating minioclient) Couldn't create bucket because: '%w'\n", err)
	}
	uploadInfo, err := minioClient.PutObject(contextBackground, userId, noteName, strings.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: "application/text"})
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewBucket() -> (Fail uploading Object) Couldn't create note because: '%w'\n", err)
	}
	return uploadInfo, nil
}
