package yana

import (
	"context" // Why does this exist????
	"fmt"
	"io"
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

type Note struct {
	PostgresId       string // TODO: Maybe make this a UUID instead of a string in the future?
	Name             string
	BucketName       string // TODO: Maybe make this a UUID instead of a string in the future?
	Content          string
	CreatedAtUTC     string
	ContentShortened string
}

type MinIOConfig struct {
	Url       string `yaml:"url"`
	AccessKey string `yaml:"accesskey"`
	SecretKey string `yaml:"secretkey"`
	UseSSL    bool   `yaml:"usessl"`
}

var yanaContext context.Context = context.Background()

const MINIO_CONFIG_PATH = "db/minio.yml"

func readMinIOConfig(path string) (MinIOConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return MinIOConfig{}, err
	}

	config := MinIOConfig{}
	yaml.Unmarshal(file, &config)
	if err != nil {
		return MinIOConfig{}, fmt.Errorf("Error in file %q: %w", path, err)
	}
	return config, err
}

var EMPTY_CLIENT = &minio.Client{}
var minioClient = EMPTY_CLIENT

// TODO: Maybe replace error with YanaError???
func checkMinIOClient() error {
	if minioClient != EMPTY_CLIENT {
		return nil
	}
	config, err := readMinIOConfig(MINIO_CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("Error in yana.generateMinIOClient (Couldn't read minio config) -> err: %w")
	}
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	}
	minioClient, err = minio.New(config.Url, options)
	if err != nil {
		return fmt.Errorf("Error in yana.generateMinIOClient (Couldn't connect to minio) -> err: %w")
	}

	return nil
}

func doesNoteWithSameNameExist(bucketName, noteName string) (bool, YanaError) {
	err := checkMinIOClient()
	if err != nil {
		return false, YanaError{Code: BadClient, Err: fmt.Errorf("Error in doesNoteWithSameNameExist: Error checking minio client: %w ", err)}
	}
	_, err = minioClient.GetObject(yanaContext, bucketName, noteName, minio.GetObjectOptions{})
	if err != nil {
		fmt.Printf("Note Already Exists: %w", err)
		return true, YanaError{Code: NoError, Err: err}
	}
	return false, YanaError{Code: NoError, Err: nil}
}

func GetAllNotesOfUser(bucketName string) ([]Note, error) {
	err := checkMinIOClient()
	if err != nil {
		return []Note{}, nil
	}
	cont := context.Background()
	objectChannel := minioClient.ListObjects(cont, bucketName, minio.ListObjectsOptions{Recursive: true})
	var notes []Note
	i := -1 // -1 so the index is 0 at the start of the for (each) loop
	for objectInfo := range objectChannel {
		i++
		if objectInfo.Err != nil {
			fmt.Printf("yana.GetAllNotesOfUser() -> Failing to get object at index %d because '%w'\n\n", i, objectInfo.Err)
			continue
		}
		actualNote, err := GetNote(bucketName, objectInfo.Key) // Might not actually work lol
		if err != nil {
			fmt.Printf("yana.GetAllNotesOfUser() -> Failing to fetch actual object at index %d because '%w'\n\n", i, err)
			continue
		}
		notes = append(notes, actualNote)
	}
	fmt.Println("notes:", notes)
	fmt.Println("notes len:", len(notes))
	fmt.Println("Last index = ", i)
	return notes, nil
}

func shortenNoteContent(content string) string {
	if len(content) >= 10 {
		return content[:6] + "..."
	}
	return content
}

func GetNote(bucketName, noteName string) (Note, error) {
	err := checkMinIOClient()
	if err != nil {
		return Note{}, fmt.Errorf("yana.GetNote() -> (Fail generating minioclient) Couldn't create minio because: '%w'\n", err)
	}
	fmt.Println("bucketName in GetNote: ", bucketName)
	fmt.Println("noteName in GetNote: ", bucketName)
	object, err := minioClient.GetObject(yanaContext, bucketName, noteName, minio.GetObjectOptions{})
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note in yana.GetNote(): %w", err)
	}
	defer object.Close()

	_, err = object.Stat()
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note metadata in yana.GetNote(): %w", err)
	}
	content, err := io.ReadAll(object)
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note content in yana.GetNote(): %w", err)
	}
	postgresNoteInfo, err := GetPostgresNoteInfo(bucketName, noteName)
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note metadata (from postgres) in yana.GetNote(): %w", err)
	}
	return Note{
		PostgresId:       postgresNoteInfo.Id,
		Name:             noteName,
		BucketName:       bucketName,
		Content:          string(content),
		CreatedAtUTC:     postgresNoteInfo.CreatedAtUTC,
		ContentShortened: shortenNoteContent(string(content))}, nil
}

func NewBucket(bucketName string) error {
	err := checkMinIOClient()
	if err != nil {
		return fmt.Errorf("yana.NewBucket() -> (Fail generating minioclient) Couldn't create bucket because: '%w'\n", err)
	}
	err = minioClient.MakeBucket(yanaContext, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("yana.NewBucket() -> Couldn't create bucket because: '%w'\n", err)
	}
	return nil
}

func NewNote(bucketName, noteName, content string) (minio.UploadInfo, error) {
	err := checkMinIOClient()
	if err != nil {
		fmt.Println("Some problem with checkMinIOClient: %w", err)
		return minio.UploadInfo{}, nil
	}
	isExisting, yanaErr := doesNoteWithSameNameExist(bucketName, noteName)
	if isExisting {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Note already exists) Checked if note with same name exists: '%w'", yanaErr.Err)
	}
	// The data is inserted to postgres first before actually saving the note to MinIO
	// because it feels a lot safer to remove a row in postgres than to remove an object in MinIO.
	// I also think that it might be faster to delete a row than an object
	// but that's just speculation
	err = insertNewNoteInPostgres(bucketName, noteName)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Fail inserting info to postgres) Couldn't add info to postgres because: %w", err)
	}
	err = checkMinIOClient()
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Fail generating minioclient) Couldn't create Client because: '%w'\n", err)
	}
	uploadInfo, err := minioClient.PutObject(yanaContext, bucketName, noteName, strings.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: "application/text"})
	if err != nil {
		deleteRowOfNote(bucketName, noteName)
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Fail uploading Object) Couldn't create note because: '%w'\n", err)
	}
	return uploadInfo, nil
}
