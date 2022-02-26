package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
)

func (db *DBT) OldestFromQueue(ctx context.Context) (order string, err error) {
	sql := "update queue set last_checked_at = default, in_handling = true\nwhere order_num in (select order_num from queue where in_handling = false order by last_checked_at limit 1)\nreturning order_num;"
	resp := db.Pool.QueryRow(ctx, sql)
	err = resp.Scan(&order)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrEmptyQueue
	}
	return order, err
}

func (db *DBT) DeferOrder(ctx context.Context, order, status string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sql := "update queue set in_handling = false where order_num = $1"
	_, err = db.Pool.Exec(ctx, sql, order)
	if err != nil {
		return err
	}

	//sql1 := "update accruals set status = $1 where order_num = $2;"
	//_, err = db.Pool.Exec(ctx, sql1, status, order)
	//if err != nil {
	//	return err
	//}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}
	return nil
}

func (db *DBT) FinalizeOrder(ctx context.Context, order, status string, accrual float64) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sql := "delete from queue where order_num = $1 returning user_id"
	resp := db.Pool.QueryRow(ctx, sql, order)
	var userID int
	err = resp.Scan(&userID)
	if err != nil {
		return err
	}

	sql1 := "update accruals set status = $1, accrual = $2 where order_num = $3"
	_, err = db.Pool.Exec(ctx, sql1, status, accrual, order)
	if err != nil {
		return err
	}

	sql2 := "update balance set available = available + $1 where user_id = $2"
	_, err = db.Pool.Exec(ctx, sql2, accrual, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}
	return nil
}
