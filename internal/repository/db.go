package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/antonevtu/go-musthave-shortener-tpl/internal/pool"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
)

type dbT struct {
	*pgxpool.Pool
}
