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

const POSTGRES_CONFIG_PATH = "db/postgres.yml"

type PostgresConfig struct {
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

func readPostgresConfig(path string) (PostgresConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return PostgresConfig{}, err
	}

	config := PostgresConfig{}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return PostgresConfig{}, fmt.Errorf("in file %q: %w", path, err)
	}
	return config, err
}

func arePasswordsSame(firstPassword string, secondPassword string) bool {
	// TODO: Check with Hashes and stuff once passwords are not just stored raw
	return firstPassword == secondPassword
}

func IsLoginOk(email string, password string) (bool, YanaError) {
	db, err := connectToPostgres()
	if err != nil {
		return false, YanaError{Code: ConnectionFailed, Err: fmt.Errorf("yana.CheckPassword() -> Couldn't connect to Postgres: %w", err)}
	}
	// TODO: Hash and salt.. this and that...
	var actualPassword string
	query := `SELECT encryptedpassword FROM user_ WHERE email = $1`
	row := db.QueryRow(query, email)
	fmt.Println("row = ", row)
	fmt.Println("row.Err() = ", row.Err())
	row.Scan(&actualPassword)
	defer db.Close()
	if row.Err() == sql.ErrNoRows || actualPassword == "" {
		fmt.Println("err = ", err)
		fmt.Printf("actualPassword = \"%s\"\n", actualPassword)
		return false, YanaError{Code: UserNotFound, Err: fmt.Errorf("yana.IsLoginOk() -> Couldn't find user")}
	} else if err != nil {
		return false, YanaError{Code: ConnectionFailed, Err: fmt.Errorf("yana.IsLoginOk() -> Couldn't execute query: %w", err)}
	} else if !arePasswordsSame(password, actualPassword) {
		fmt.Printf("pasword input: \"%s\"\n", password)
		fmt.Printf("actualPasword: \"%s\"\n", actualPassword)
		return false, YanaError{Code: PasswordsNotEqual, Err: fmt.Errorf("yana.IsLoginOk() -> Passwords are not equal")}
	}
	return true, YanaError{}
}

func GetUserIDFromEmail(email string) (string, error) {
	db, err := connectToPostgres()
	defer db.Close()
	if err != nil {
		return "", fmt.Errorf("yana.GetUserIDFromEmail() -> Couldn't connect to Postgres: %w", err)
	}
	var userid string
	query := `SELECT id FROM user_ WHERE email = $1`
	row := db.QueryRow(query, email)
	row.Scan(&userid)
	if err == sql.ErrNoRows || userid == "" {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("yana.GetUserIDFromEmail() -> Couldn't execue qurey: %w", err)
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

func connectToPostgres() (*sql.DB, error) {
	config, err := readPostgresConfig(POSTGRES_CONFIG_PATH)
	if err != nil {
		return &sql.DB{}, fmt.Errorf("yana.connectToPostgres() -> Couldn't load postgres config: %w", err)
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DatabaseName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		defer db.Close()
		return &sql.DB{}, fmt.Errorf("yana.connectToPostgres() -> Couldn't connect to postgres: %w", err)
	}
	err = db.Ping()
	if err != nil {
		defer db.Close()
		return &sql.DB{}, fmt.Errorf("yana.connectToPostgres() -> Couldn't verify connection to postgres: %w", err)
	}
	return db, nil
}

func isUserInDatabase(email string) (bool, error) {
	db, err := connectToPostgres()
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

func NewUserNoUsername(email string, password string) (string, error) {
	address, errIsEmailValid := mail.ParseAddress(email)
	if errIsEmailValid != nil {
		return "", errIsEmailValid
	}
	return InsertNewUserInPostgres(email, address.Name, password)
}

// Returns string: uuid of newly created user
func InsertNewUserInPostgres(email string, fullname string, password string) (string, error) {
	_, errIsEmailValid := mail.ParseAddress(email)
	if errIsEmailValid != nil {
		return "", errIsEmailValid
	}
	db, err := connectToPostgres()
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

	sql := `INSERT INTO user_ (id, fullname, encryptedpassword ,email) VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(sql, userid, fullname, encryptedPassword, email)
	if err != nil {
		return "", fmt.Errorf("yana.InsertNewUserInPostgres() -> Insert query wasn't succesful: %w", err)
	}
	return userid, nil
}
