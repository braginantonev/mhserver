package auth

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	INSERT_USER   string = "INSERT INTO users (user, password) VALUES (?, ?)"
	SELECT_USERID string = "SELECT id FROM users WHERE user = ?"
	SELECT_USER   string = "SELECT user, password FROM users WHERE user = ?"
)

type User struct {
	Name     string `json:"user"`
	Password string `json:"pass"`
}

func NewUser(name string, password string) User {
	return User{
		Name:     name,
		Password: password,
	}
}

// If user exist in database, return personal jwt token
func Login(user User, db *sql.DB, jwt_signature string) (string, error) {
	if user.Name == "" {
		return "", ErrNameIsEmpty
	}

	db_user := User{}
	row := db.QueryRow(SELECT_USER, user.Name)
	if err := row.Scan(&db_user.Name, &db_user.Password); err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotExist
		}

		return "", fmt.Errorf("%w: %s", ErrInternal, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(db_user.Password), []byte(user.Password)); err != nil {
		return "", ErrWrongPassword
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": user.Name,
		"nbf":  now.Unix(),
		"exp":  now.Add(24 * time.Hour).Unix(),
		"iat":  now.Unix(),
	})

	token_str, err := token.SignedString([]byte(jwt_signature))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInternal, err)
	}

	return token_str, nil
}

// Crypt user password and put them to database
func Register(user User, db *sql.DB) error {
	if user.Name == "" {
		return ErrNameIsEmpty
	}

	row := db.QueryRow(SELECT_USERID, user.Name)
	if err := row.Scan(); err != sql.ErrNoRows {
		return ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInternal, err)
	}

	if _, err = db.Exec(INSERT_USER, user.Name, string(hash)); err != nil {
		return fmt.Errorf("%w: %s", ErrInternal, err)
	}

	return nil
}

func CheckJWTUserMatch(username string, token string, signature string) error {
	tokenFromString, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %s", ErrJwtSignatureInvalid, token.Header["alg"])
		}

		return []byte(signature), nil
	})

	if err != nil {
		return err
	}

	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		if claims["name"] != username {
			return fmt.Errorf("%w\n\texpected user name: `%s`, but got `%s`", ErrWrongJWTName, username, claims["name"])
		}
	} else {
		return ErrBadClaims
	}

	return nil
}
