package yana

import (
	"database/sql"
	"fmt"
	"net/mail"
	"os"

	// "golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

// I am willingly ignoring Golang's styleguide for constants
// https://google.github.io/styleguide/go/guide#mixedcaps
const (
	EMAIL_LOCAL_MAX_LEN  = 64
	EMAIL_DOMAIN_MAX_LEN = 255
	EMAIL_MAX_LEN        = 320
)

const POSTGRESQL_CONFIG_PATH = "config/postgresql.yml"

type PostgreSQLConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	DatabaseName string `yaml:"db"`
	Password     string `yaml:"password"`
}

type User struct {
	UserId   string
	Email    string
	FullName string
}

type PostgreSQLNote struct {
	Id           string
	Bucketname   string
	Filename     string
	CreatedAtUTC string
}

func readPostgreSQLConfig(path string) (PostgreSQLConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return PostgreSQLConfig{}, err
	}

	config := PostgreSQLConfig{}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return PostgreSQLConfig{}, fmt.Errorf("in file %q: %w", path, err)
	}
	return config, err
}

func arePasswordsSame(firstPassword string, secondPassword string) bool {
	// TODO: Check with Hashes and stuff once passwords are not just stored raw
	return firstPassword == secondPassword
}

func IsLoginOk(email string, password string) (bool, YanaError) {
	db, err := connectToPostgreSQL()
	if err != nil {
		return false, YanaError{Code: ConnectionFailed, Err: fmt.Errorf("yana.CheckPassword() -> Couldn't connect to Postgres: %w", err)}
	}
	var actualPassword string
	query := `SELECT encryptedpassword FROM user_ WHERE email = $1`
	row := db.QueryRow(query, email)
	row.Scan(&actualPassword)
	defer db.Close()
	if row.Err() == sql.ErrNoRows || actualPassword == "" {
		return false, YanaError{Code: UserNotFound, Err: fmt.Errorf("yana.IsLoginOk() -> Couldn't find user")}
	} else if err != nil {
		return false, YanaError{Code: ConnectionFailed, Err: fmt.Errorf("yana.IsLoginOk() -> Couldn't execute query: %w", err)}
	} else if !arePasswordsSame(password, actualPassword) {
		return false, YanaError{Code: PasswordsNotEqual, Err: fmt.Errorf("yana.IsLoginOk() -> Passwords are not equal")}
	}
	return true, YanaError{}
}

func GetUserIDFromEmail(email string) (string, error) {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return "", fmt.Errorf("yana.GetUserIDFromEmail() -> Couldn't connect to PostgreSQL: %w", err)
	}
	var userid string
	query := `SELECT id FROM user_ WHERE email = $1`
	row := db.QueryRow(query, email)
	row.Scan(&userid)
	if err == sql.ErrNoRows || userid == "" {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("yana.GetUserIDFromEmail() -> Couldn't execue query: %w", err)
	}
	return userid, nil
}

func GetUserFromUserID(userid string) (User, error) {
	return User{}, nil // NOTE: Implement if necessary
}

func generateUserID() string {
	return uuid.New().String()
}

// Future TODO
func hashAndSalt(password []byte) (string, error) {
	// NOTE: Authentication will be dealt with later...
	// For now, passwords are transmitted and stored without any encryption, SSL, hashing, ...

	// hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	// if err != nil {
	// 	log.Printf("Error happened in encryptPassword: %w", err)
	// 	return "", err
	// }
	// return string(hash), nil

	return "", nil
}

func connectToPostgreSQL() (*sql.DB, error) {
	config, err := readPostgreSQLConfig(POSTGRESQL_CONFIG_PATH)
	if err != nil {
		return &sql.DB{}, fmt.Errorf("yana.connectToPostgreSQL() -> Couldn't load postgresql config: %w", err)
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DatabaseName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		defer db.Close()
		return &sql.DB{}, fmt.Errorf("yana.connectToPostgreSQL() -> Couldn't connect to postgres: %w", err)
	}
	err = db.Ping()
	if err != nil {
		defer db.Close()
		return &sql.DB{}, fmt.Errorf("yana.connectToPostgreSQL() -> Couldn't verify connection to postgres: %w", err)
	}
	return db, nil
}

func isUserInDatabase(email string) (bool, error) {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return false, fmt.Errorf("yana.checkIfUserExists() -> Couldn't connect to Postgres: %w", err)
	}
	var id string
	query := `SELECT id FROM user_ WHERE email = $1`
	row := db.QueryRow(query, email)
	row.Scan(&id)
	if err == sql.ErrNoRows || id == "" {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("yana.isUserInDatabase() -> Couldn't execue qurey: %w", err)
	}
	return true, nil
}

// This has been here since before I even had a frontend and never needed it
// because the email field in /register is required
// func NewUserNoUsername(email string, password string) (string, error) {
// 	address, errIsEmailValid := mail.ParseAddress(email)
// 	if errIsEmailValid != nil {
// 		return "", errIsEmailValid
// 	}
// 	return CreateNewUser(email, address.Name, password)
// }

// Returns string: uuid of newly created user
func CreateNewUser(email string, fullname string, password string) (string, error) {
	_, errIsEmailValid := mail.ParseAddress(email)
	if errIsEmailValid != nil {
		return "", errIsEmailValid
	}
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return "", err
	}

	isUserInDB, err := isUserInDatabase(email)
	if err != nil {
		return "", err
	}
	if isUserInDB {
		return "", nil
	}

	userid := generateUserID()

	// TODO: Encrypt Password
	encryptedPassword := password

	query := `INSERT INTO user_ (id, fullname, encryptedpassword, email) VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(query, userid, fullname, encryptedPassword, email)
	if err != nil {
		return "", fmt.Errorf("yana.CreateNewUser() -> Insert query wasn't succesful: %w", err)
	}
	return userid, nil
}

// err == nil means deletion was  succesful
func deleteRowOfNote(bucketName, fileName string) error {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return err
	}
	query := "DELETE FROM note WHERE bucketname = $1 AND filename = $2"
	_, err = db.Exec(query, bucketName, fileName)
	if err != nil {
		return fmt.Errorf("Error in yana.deleteRowOfNote() -> Couldn't delete row because: %x", err)
	}
	return nil
}

func insertNewNoteInPostgreSQL(bucketName, filename string) error {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return fmt.Errorf("Error in yana.insertNewNoteInPostgreSQL() -> couldn't create to postgresql because: %x", err)
	}

	isNoteInDB, yanaErr := doesNoteWithSameNameExist(bucketName, filename)
	if yanaErr.Err != nil {
		return fmt.Errorf("Error in yana.insertNewNoteInPostgreSQL() -> Note in user bucket with same name exists: %w", yanaErr.Err)
	}
	if isNoteInDB {
		return fmt.Errorf("Error in yana.insertNewNoteInPostgreSQL() -> Note with same name is already in user's bucket")
	}

	query := `INSERT INTO note (id, bucketname, filename, created_at_utc) VALUES (gen_random_uuid(), $1, $2, timezone('utc', NOW()::timestamp))`
	_, err = db.Exec(query, bucketName, filename)
	if err != nil {
		return fmt.Errorf("Error in yana.insertNewNoteInPostgreSQL() -> Insert query wasn't succesful: %w", err)
	}

	return nil
}

func insertNoteInPostgreSQL(noteId, bucketName, filename, creationDateUTC string) error {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return fmt.Errorf("Error in yana.insertNoteInPostgreSQL() -> couldn't create to postgresql because: %x", err)
	}
	query := `INSERT INTO note (id, bucketname, filename, created_at_utc) VALUES ($1, $2, $3, %4)`
	_, err = db.Exec(query, bucketName, filename)
	if err != nil {
		return fmt.Errorf("Error in yana.insertNoteInPostgreSQL() -> Insert query wasn't succesful: %w", err)
	}

	return nil
}

func getPostgreSQLNoteFromBucketAndNotename(bucketname, filename string) (PostgreSQLNote, error) {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return PostgreSQLNote{}, fmt.Errorf("Error in yana.getPostgreSQLNoteFromBucketAndNotename() -> couldn't create to postgresql because: %x", err)
	}

	var id string
	var bucketnameFromPostgreSQL string
	var filenameFromPostgreSQL string
	var creationDate string
	query := `SELECT id, bucketname, filename, created_at_utc FROM note WHERE bucketname = $1 AND filename = $2`
	err = db.QueryRow(query, bucketname, filename).Scan(&id, &bucketnameFromPostgreSQL, &filenameFromPostgreSQL, &creationDate)
	if err != nil {
		return PostgreSQLNote{}, fmt.Errorf("Error in yana.getPostgreSQLNoteFromBucketAndNotename() -> Select query wasn't succesful: %w", err)
	}
	return PostgreSQLNote{id, bucketnameFromPostgreSQL, filenameFromPostgreSQL, creationDate}, nil
}

func getPostgreSQLNoteFromNoteId(postgresNoteId string) (PostgreSQLNote, error) {
	db, err := connectToPostgreSQL()
	defer db.Close()
	if err != nil {
		return PostgreSQLNote{}, fmt.Errorf("Error in yana.getPostgreSQLNoteFromNoteId() -> couldn't create to postgresql because: %x", err)
	}

	var bucketname string
	var filename string
	var creationDate string
	query := `SELECT bucketname, filename, created_at_utc FROM note WHERE id = $1`
	err = db.QueryRow(query, postgresNoteId).Scan(&bucketname, &filename, &creationDate)
	if err != nil {
		return PostgreSQLNote{}, fmt.Errorf("Error in yana.getPostgreSQLNoteFromNoteId() -> Select query wasn't succesful: %w", err)
	}
	return PostgreSQLNote{postgresNoteId, bucketname, filename, creationDate}, nil
}

func updateNoteNameInPostgreSQL(noteId, newNoteName string) error {
	db, err := connectToPostgreSQL()
	if err != nil {
		fmt.Errorf("Error in yana.updateNoteNameInPostgreSQL -> Couldn't connect to postgresql because '%w'", err)
	}
	defer db.Close()
	query := `UPDATE note SET filename=$1 WHERE id=$2`
	_, err = db.Exec(query, newNoteName, noteId)
	if err != nil {
		fmt.Errorf("Error in yana.updateNoteNameInPostgreSQL -> Couldn't execute update query because '%w'", err)
	}
	return nil
}

func deleteNoteInPostgres(noteId string) error {
	db, err := connectToPostgreSQL()
	if err != nil {
		fmt.Errorf("Error in yana.deleteNoteInPostgres() -> Couldn't connect to postgresql because '%w'", err)
	}
	defer db.Close()
	query := `DELETE FROM note WHERE id=$1`
	_, err = db.Exec(query, noteId)
	if err != nil {
		fmt.Errorf("Error in yana.deleteNoteInPostgres() -> Couldn't execute delete query because '%w'", err)
	}
	return nil
}

// For editing an already existing note
func doesOtherNoteWithSameNameExist(noteId, bucketName, filename string) (bool, error) {
	db, err := connectToPostgreSQL()
	if err != nil {
		fmt.Errorf("Error in yana.doesOtherNoteWithSameNameExist() -> Couldn't connect to postgresql because '%w'", err)
	}
	defer db.Close()
	var unusedId string
	query := `SELECT id FROM note WHERE id!=$1 AND bucketname=$2 AND filename=$3`
	err = db.QueryRow(query, noteId, bucketName, filename).Scan(&unusedId)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("Error in yana.doesOtherNoteWithSameNameExist() -> Couldn't execute querye to check if a different note with the same name already exists because '%w'", err)
	}
	return true, nil
}
