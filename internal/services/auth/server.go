package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/braginantonev/mhserver/internal/repository"
	pb "github.com/braginantonev/mhserver/proto/gen/auth"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	USER_NAME_MAX_LENGTH int = 30

	SELECT_USERID              string = "SELECT id FROM users WHERE user = ?"
	SELECT_USER_PASS           string = "SELECT password FROM users WHERE user = ?"
	SELECT_REGISTER_SECRET_KEY string = "SELECT id FROM register_secret_keys WHERE secret_key = ?"

	INSERT_REG_USER string = "INSERT INTO users (user, password) VALUES (?, ?)"
	DELETE_REG_KEY  string = "DELETE FROM register_secret_keys WHERE id = ?"

	DATABASE_TIMEOUT time.Duration = 2 * time.Second
)

type AuthServer struct {
	pb.AuthServiceServer
	cfg AuthConfig
	sem repository.Semaphore
	db  *sql.DB
}

func NewAuthServer(cfg AuthConfig, db *sql.DB) *AuthServer {
	return &AuthServer{
		cfg: cfg,
		sem: repository.NewSemaphore(100),
		db:  db,
	}
}

func (s *AuthServer) Login(ctx context.Context, user *pb.User) (*pb.LoginResponse, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	var saved_pass_hash string
	row := s.db.QueryRow(SELECT_USER_PASS, user.Name)
	if err := row.Scan(&saved_pass_hash); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotExist
		}

		slog.ErrorContext(ctx, "failed scan sql rows for login request", slog.Any("err", err))
		return nil, ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(saved_pass_hash), []byte(user.Password)); err != nil {
		return nil, ErrWrongPassword
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": user.Name,
		"nbf":  now.Unix(),
		"exp":  now.Add(24 * time.Hour).Unix(),
		"iat":  now.Unix(),
	})

	token_str, err := token.SignedString([]byte(s.cfg.JWTSignature))
	if err != nil {
		slog.ErrorContext(ctx, "failed complete signed jwt token", slog.Any("err", err))
		return nil, ErrInternal
	}

	return &pb.LoginResponse{Token: token_str}, nil
}

func (s *AuthServer) Register(ctx context.Context, reg_info *pb.RegisterRequest) (*emptypb.Empty, error) {
	s.sem.Acquire()
	defer s.sem.Release()

	if reg_info == nil {
		return nil, ErrEmptyRequest
	}

	username := reg_info.User.GetName()

	if len(username) == 0 {
		return nil, ErrNullUsername
	}

	if len(username) > USER_NAME_MAX_LENGTH {
		return nil, ErrNameTooLong
	}

	db_ctx, cancel := context.WithTimeout(ctx, DATABASE_TIMEOUT)
	defer cancel()

	row := s.db.QueryRowContext(db_ctx, SELECT_USERID, username)
	if err := row.Scan(); err == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		slog.ErrorContext(ctx, "failed select user from database", slog.Any("error", err))
		return nil, ErrInternal
	}

	db_ctx, cancel = context.WithTimeout(ctx, DATABASE_TIMEOUT)
	defer cancel()

	if len(reg_info.SecretKey) == 0 {
		return nil, ErrRegSecretKeyNotFound
	}

	var key_id int
	key_row := s.db.QueryRowContext(db_ctx, SELECT_REGISTER_SECRET_KEY, reg_info.SecretKey)
	if err := key_row.Scan(&key_id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRegSecretKeyNotFound
		}

		slog.ErrorContext(ctx, "failed select reg key from database", slog.Any("error", err))
		return nil, ErrInternal
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(reg_info.User.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "failed generate hash from password", slog.Any("err", err))
		return nil, ErrInternal
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed start register user transaction", slog.Any("error", err))
		return nil, ErrInternal
	}

	db_ctx, cancel = context.WithTimeout(ctx, DATABASE_TIMEOUT)
	defer cancel()

	if _, err = tx.ExecContext(db_ctx, INSERT_REG_USER, username, string(hash)); err != nil {
		slog.ErrorContext(ctx, "failed insert user to database", slog.Any("err", err))
		return nil, ErrInternal
	}

	db_ctx, cancel = context.WithTimeout(ctx, DATABASE_TIMEOUT)
	defer cancel()

	if _, err = tx.ExecContext(db_ctx, DELETE_REG_KEY, key_id); err != nil {
		slog.ErrorContext(ctx, "failed insert user to database", slog.Any("err", err))
		return nil, ErrInternal
	}

	if err = tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "failed commit register db transaction", slog.Any("err", err))
		return nil, ErrInternal
	}

	return &emptypb.Empty{}, nil
}
