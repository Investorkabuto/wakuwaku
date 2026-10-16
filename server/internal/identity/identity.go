// Package identity は利用者アカウントを管理する。
package identity

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Resolve は auth_uid に対応するユーザーIDを返す。未登録なら作成する。
func (s *Store) Resolve(ctx context.Context, authUID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (auth_uid) VALUES ($1)
		ON CONFLICT (auth_uid) DO UPDATE SET auth_uid = EXCLUDED.auth_uid
		RETURNING id::text`, authUID).Scan(&id)
	return id, err
}
