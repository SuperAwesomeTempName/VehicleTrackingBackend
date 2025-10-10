package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func Connect(ctx context.Context, dsn string) error {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return err
	}
	p, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	pool = p
	return nil
}

func Close() {
	if pool != nil {
		pool.Close()
	}
}

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
}

func InsertUser(ctx context.Context, id, name, email, passwordHash string) error {
	_, err := pool.Exec(ctx, `INSERT INTO users (id,name,email,password_hash) VALUES ($1,$2,$3,$4)`, id, name, email, passwordHash)
	return err
}

func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	row := pool.QueryRow(ctx, `SELECT id,name,email,password_hash,role FROM users WHERE email=$1`, email)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role); err != nil {
		return nil, err
	}
	return u, nil
}
func Ping(ctx context.Context) error {
	if pool == nil {
		return fmt.Errorf("no db pool initialized")
	}
	return pool.Ping(ctx)
}

// GetUserByID returns user by id
func GetUserByID(ctx context.Context, id string) (*User, error) {
	row := pool.QueryRow(ctx, `SELECT id,name,email,password_hash,role FROM users WHERE id=$1`, id)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role); err != nil {
		return nil, err
	}
	return u, nil
}

// StoreRefreshToken stores a refresh token hash for a user
func StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := pool.Exec(ctx, `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1,$2,$3)`, userID, tokenHash, expiresAt)
	return err
}

// GetUserIDByRefreshHash returns (userID, ok, error)
func GetUserIDByRefreshHash(ctx context.Context, tokenHash string) (string, bool, error) {
	row := pool.QueryRow(ctx, `SELECT user_id FROM refresh_tokens WHERE token_hash=$1 AND revoked=false AND expires_at > now() LIMIT 1`, tokenHash)
	var userID string
	if err := row.Scan(&userID); err != nil {
		return "", false, err
	}
	return userID, true, nil
}

// RevokeRefreshTokenByHash marks token revoked
func RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	_, err := pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE token_hash=$1`, tokenHash)
	return err
}
