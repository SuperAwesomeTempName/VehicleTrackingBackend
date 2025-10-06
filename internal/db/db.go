package db

import (
	"context"

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
