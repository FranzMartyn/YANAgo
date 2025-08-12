package yana

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"
)

const (
	FILENAME_MAX_LEN             = 1024 // See https://min.io/docs/minio/windows/operations/concepts/thresholds.html#:~:text=Maximum%20length%20for%20object%20names
	BUCKETNAME_MAX_LEN           = 63
	DEFAULT_BUCKET_SERVER_REGION = "us-east-1" // See https://min.io/docs/minio/linux/developers/go/API.html#MakeBucket:~:text=(defaults%20to%20us%2Deast%2D1).
)

type Note struct {
	PostgreSQLId     string // TODO: Maybe make this a UUID instead of a string in the future?
	Name             string
	BucketName       string // TODO: Maybe make this a UUID instead of a string in the future?
	Content          string
	CreatedAtUTC     string
	ContentShortened string
}

type UpdatedNoteState struct {
	State int
}

const (
	NewNoteState = iota
	NothingHappenedState
	OldNoteState
	NoteDeletedState
)

type MinIOConfig struct {
	Url       string `yaml:"url"`
	AccessKey string `yaml:"accesskey"`
	SecretKey string `yaml:"secretkey"`
	UseSSL    bool   `yaml:"usessl"`
}

var yanaContext context.Context = context.Background()

const MINIO_CONFIG_PATH = "config/minio.yml"

// Just for myself/the developer to have an easy to time to print the error
func (updatedNoteState UpdatedNoteState) ToString() string {
	switch updatedNoteState.State {
	case NewNoteState:
		return "NewNoteState: Succesfully updated note"
	case NothingHappenedState:
		return "NothingHappenedState: Note couldn't be updated and is (still) in it's original state"
	case OldNoteState:
		return "OldNoteState: Note couldn't be updated and is back to it's original state"
	case NoteDeletedState:
		return "NoteDeletedState: Note couldn't be updated and has been unfortunately deleted"
	}
	return ""
}

// Turns out that unix-like systems support a plethora of characters.
// index.html already makes the input <= 1024 and filters out NUL and /, but
// checking here too because you can't trust the user
func isFilenameOk(filename string) bool {
	containsNULCharacter := strings.ContainsRune(filename, '\x00')
	containsSlash := strings.ContainsRune(filename, '/')    // Can't escape / in a file.
	isDotOrDotDot := filename == "." || filename == ".."    // I don't know a better name for this variable.
	isLongerThanAllowed := len(filename) > FILENAME_MAX_LEN // Only limited by the S3 Api.
	return !containsNULCharacter && !containsSlash && !isLongerThanAllowed && !isDotOrDotDot
}

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

/*
 * This function should be called at the start
 * of every function that is using minioClient
 */
func checkMinIOClient() error {
	// TODO: Maybe replace error with YanaError???
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
		return false, YanaError{Code: BadClient, Err: fmt.Errorf("Error in yana.doesNoteWithSameNameExist(): Error checking minio client: %w ", err)}
	}
	object, err := minioClient.GetObject(yanaContext, bucketName, noteName, minio.GetObjectOptions{})
	if err != nil {
		return false, YanaError{Code: NoError, Err: nil}
	}
	defer object.Close()
	_, err = object.Stat()
	if err != nil {
		return false, YanaError{Code: NoError, Err: nil}
	}
	return true, YanaError{Code: NoError, Err: nil}
}

func GetAllNotesOfUser(bucketName string) ([]Note, error) {
	err := checkMinIOClient()
	if err != nil {
		return []Note{}, nil
	}
	objectChannel := minioClient.ListObjects(yanaContext, bucketName, minio.ListObjectsOptions{Recursive: true})
	var notes []Note
	for objectInfo := range objectChannel {
		if objectInfo.Err != nil {
			continue
		}
		actualNote, err := GetNoteFromBucketAndNotename(bucketName, objectInfo.Key) // Might not actually work lol
		if err != nil {
			continue
		}
		notes = append(notes, actualNote)
	}
	return notes, nil
}

func shortenNoteContent(content string) string {
	if len(content) >= 25 {
		return content[:21] + "..."
	}
	return content
}

func GetNoteFromBucketAndNotename(bucketName, noteName string) (Note, error) {
	if !isFilenameOk(noteName) {
		return Note{}, fmt.Errorf("Error in yana.GetNoteFromBucketAndNotename(): Filename is not ok")
	}

	err := checkMinIOClient()
	if err != nil {
		return Note{}, fmt.Errorf("yana.GetNoteFromBucketAndNotename() -> Couldn't create minio because: '%w'\n", err)
	}
	object, err := minioClient.GetObject(yanaContext, bucketName, noteName, minio.GetObjectOptions{})
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note in yana.GetNoteFromBucketAndNotename(): %w", err)
	}
	defer object.Close()

	content, err := io.ReadAll(object)
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note content in yana.GetNoteFromBucketAndNotename(): %w", err)
	}
	postgresqlNoteInfo, err := getPostgreSQLNoteFromBucketAndNotename(bucketName, noteName)
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note metadata (from postgresql) in yana.GetNoteFromBucketAndNotename(): %w", err)
	}
	return Note{
		PostgreSQLId:     postgresqlNoteInfo.Id,
		Name:             noteName,
		BucketName:       bucketName,
		Content:          string(content),
		CreatedAtUTC:     postgresqlNoteInfo.CreatedAtUTC,
		ContentShortened: shortenNoteContent(string(content))}, nil
}

func GetNoteFromNoteId(postgresqlNoteId string) (Note, error) {
	err := checkMinIOClient()
	if err != nil {
		return Note{}, fmt.Errorf("yana.GetNoteFromNoteId() -> (Fail generating minioclient) Couldn't create minio because: '%w'\n", err)
	}
	postgresqlNoteInfo, err := getPostgreSQLNoteFromNoteId(postgresqlNoteId)
	if err != nil {
		return Note{}, fmt.Errorf("yana.GetNoteFromNoteId() -> (Fail getting postgresqlNoteInfo) Couldn't get postgreSQLNoteInfo: '%w'\n", err)
	}

	// I am not using GetNoteFromBucketAndNotename() and do a normal GetObject call here because otherwise I would have a second unnecessary call to postgresql
	object, err := minioClient.GetObject(yanaContext, postgresqlNoteInfo.Bucketname, postgresqlNoteInfo.Filename, minio.GetObjectOptions{})
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note in yana.GetNoteFromNoteId(): %w", err)
	}
	defer object.Close()
	raw_content, err := io.ReadAll(object)
	if err != nil {
		return Note{}, fmt.Errorf("Couldn't get note content in yana.GetNoteFromNoteId(): %w", err)
	}
	content := string(raw_content)
	return Note{
		PostgreSQLId:     postgresqlNoteInfo.Id,
		Name:             postgresqlNoteInfo.Filename,
		BucketName:       postgresqlNoteInfo.Bucketname,
		Content:          content,
		CreatedAtUTC:     postgresqlNoteInfo.CreatedAtUTC,
		ContentShortened: shortenNoteContent(content)}, nil
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
	if content == "error" {
		return minio.UploadInfo{}, fmt.Errorf("content is not allowed to just be \"error\"")
	}
	if !isFilenameOk(noteName) {
		return minio.UploadInfo{}, fmt.Errorf("Error in yana.NewNote(): Filename is not ok")
	}

	err := checkMinIOClient()
	if err != nil {
		return minio.UploadInfo{}, nil
	}

	fmt.Println("noteName=", noteName)
	isExisting, yanaErr := doesNoteWithSameNameExist(bucketName, noteName)
	if isExisting {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Note already exists) A note with the same name already exists")
	}
	if yanaErr.Err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> Couldn't check if note with same name exists: '%w'", yanaErr.Err)
	}

	// The data is inserted to postgresql first before actually saving the note to MinIO
	// because it feels a lot safer to remove a row in postgresql than to remove an object in MinIO.
	// I also think that it might be faster to delete a row than an object
	// but that's just speculation
	err = insertNewNoteInPostgreSQL(bucketName, noteName)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Fail inserting info to postgres) Couldn't add info to postgresql because: %w", err)
	}
	err = checkMinIOClient()
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Fail generating minioclient) Couldn't create Client because: '%w'\n", err)
	}
	uploadInfo, err := minioClient.PutObject(yanaContext, bucketName, noteName, strings.NewReader(content),
		int64(len(content)), minio.PutObjectOptions{})
	if err != nil {
		deleteRowOfNote(bucketName, noteName)
		return minio.UploadInfo{}, fmt.Errorf("yana.NewNote() -> (Fail uploading Object) Couldn't create note because: '%w'\n", err)
	}
	return uploadInfo, nil
}

func UpdateNote(bucketName, noteId, newNoteName, newContent string) (UpdatedNoteState, error) {
	if !isFilenameOk(newNoteName) {
		return UpdatedNoteState{NothingHappenedState}, fmt.Errorf("Error in yana.UpdateNote(): Filename is not ok")
	}

	noteWithSameNameExist, err := doesOtherNoteWithSameNameExist(noteId, bucketName, newNoteName)
	if err != nil {
		return UpdatedNoteState{NothingHappenedState}, fmt.Errorf("Error in yana.UpdateNote(): Couldn't check if a note with the same name exists because '%w'", err)
	}
	if noteWithSameNameExist {
		return UpdatedNoteState{NothingHappenedState}, fmt.Errorf("Error in yana.UpdateNote(): A different note with the same name already exists")
	}

	oldNote, err := GetNoteFromNoteId(noteId)
	if err != nil {
		return UpdatedNoteState{NothingHappenedState}, fmt.Errorf("Error in yana.UpdateNote() -> Couldn't fetch note because: '%w'\n", err)
	}
	oldNoteName := oldNote.Name
	oldContent := oldNote.Content

	isNameChanged := oldNoteName != newNoteName
	isContentChanged := oldContent != newContent
	if !isNameChanged && !isContentChanged {
		// Not an error because the user hasn't changed anything then
		return UpdatedNoteState{NothingHappenedState}, nil
	}

	if isNameChanged {
		updateNoteNameInPostgreSQL(noteId, newNoteName)
	}
	err = checkMinIOClient()
	if err != nil {
		return UpdatedNoteState{NothingHappenedState}, fmt.Errorf("Error in yana.UpdateNote() -> Couldn't create or check minio client because: '%w'\n", err)
	}
	// At this point, if either the name or content are changed, a new file has to be created either way instead of just modifying one aspect of it...
	// Thanks MinIO devs :)
	err = minioClient.RemoveObject(yanaContext, bucketName, oldNoteName, minio.RemoveObjectOptions{})
	if err != nil {
		return UpdatedNoteState{NothingHappenedState}, fmt.Errorf("Error in yana.UpdateNote() -> Couldn't remove file to replace it because: '%w'\n", err)
	}

	// This is the scary part: The original file has been removed, but what if the file can't be created with newNoteName and newContent?
	_, newNoteErr := minioClient.PutObject(yanaContext, bucketName, newNoteName, strings.NewReader(newContent),
		int64(len(newContent)), minio.PutObjectOptions{})
	if err != nil {
		// Create a new file with the old information
		_, oldNoteErr := minioClient.PutObject(yanaContext, bucketName, oldNoteName, strings.NewReader(oldContent),
			int64(len(oldContent)), minio.PutObjectOptions{})
		if err != nil {
			// This is a horrible state: The file has been deleted in minio but a new file couldn't be created at all
			errString := "Error in yana.UpdateNote() -> Couldn't create note with either new or old information " +
				"because: '%w' with the new information and '%w' with the old information. Sorry :("
			return UpdatedNoteState{NoteDeletedState}, fmt.Errorf(errString, newNoteErr, oldNoteErr)
		}
		return UpdatedNoteState{OldNoteState}, nil
	}
	return UpdatedNoteState{NewNoteState}, nil

}

func DeleteNoteFromNoteId(noteId string) error {
	err := checkMinIOClient()
	if err != nil {
		return fmt.Errorf("Error in yana.DeleteNoteFromNoteId() -> Couldn't create or check minio client because: '%w'\n", err)
	}

	// Pretty similiar to UpdateNote()
	// 1. Try to delete note info in postgres
	// 2. Try to delete the note object in minio
	// 	  2.1 If 2. wasn't succesful, try to re-insert the data into postgres
	// 	  2.2 If 2.1 wasn't succesful, say sorry
	postgresqlNote, err := getPostgreSQLNoteFromNoteId(noteId)
	if err != nil {
		return fmt.Errorf("yana.DeleteNoteFromNoteId() -> Couldn't get Info from Postgres: '%w'\n", err)
	}

	err = deleteNoteInPostgres(noteId)
	if err != nil {
		return fmt.Errorf("yana.DeleteNoteFromNoteId() -> Couldn't delete note in Postgres: '%w'\n", err)
	}

	err = minioClient.RemoveObject(yanaContext, postgresqlNote.Bucketname, postgresqlNote.Filename, minio.RemoveObjectOptions{})
	if err != nil {
		err = insertNoteInPostgreSQL(noteId, postgresqlNote.Bucketname, postgresqlNote.Filename, postgresqlNote.CreatedAtUTC)
		if err != nil {
			// This state is BAD
			return fmt.Errorf("yana.DeleteNoteFromNoteId() -> Couldn't remove note in MinIO, but couldn't re-insert data in PostgreSQL. I'm sorry :(  :'%w'\n", err)
		}
		return fmt.Errorf("yana.DeleteNoteFromNoteId() -> Couldn't remove note in MinIO: '%w'\n", err)
	}
	return nil
}
