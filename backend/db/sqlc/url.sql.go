// Code generated by sqlc. DO NOT EDIT.
// source: url.sql

package db

import (
	"context"
)

const createUrl = `-- name: CreateUrl :one
INSERT INTO url (id, short_url, long_url)
VALUES ($1, $2, $3)
RETURNING id, short_url, long_url, created_at
`

type CreateUrlParams struct {
	ID       int64  `json:"id"`
	ShortUrl string `json:"short_url"`
	LongUrl  string `json:"long_url"`
}

func (q *Queries) CreateUrl(ctx context.Context, arg CreateUrlParams) (Url, error) {
	row := q.db.QueryRowContext(ctx, createUrl, arg.ID, arg.ShortUrl, arg.LongUrl)
	var i Url
	err := row.Scan(
		&i.ID,
		&i.ShortUrl,
		&i.LongUrl,
		&i.CreatedAt,
	)
	return i, err
}

const getUrl = `-- name: GetUrl :one
SELECT id, short_url, long_url, created_at
FROM url
WHERE short_url = $1
LIMIT 1
`

func (q *Queries) GetUrl(ctx context.Context, shortUrl string) (Url, error) {
	row := q.db.QueryRowContext(ctx, getUrl, shortUrl)
	var i Url
	err := row.Scan(
		&i.ID,
		&i.ShortUrl,
		&i.LongUrl,
		&i.CreatedAt,
	)
	return i, err
}
