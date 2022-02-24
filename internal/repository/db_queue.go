package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
)

func (db *DbT) OldestFromQueue(ctx context.Context) (order string, err error) {
	sql := "update queue set last_checked_at = default, in_handling = true\nwhere order_num in\n      (select order_num from queue order by last_checked_at limit 1) and in_handling = false\nreturning order_num;"
	resp := db.Pool.QueryRow(ctx, sql)
	err = resp.Scan(&order)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrEmptyQueue
	}
	return order, err
}

func (db *DbT) DeferOrder(ctx context.Context, order, status string) error {

	return nil
}

func (db *DbT) FinalizeOrder(ctx context.Context, order, status string) error {

	return nil
}
