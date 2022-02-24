package repository

import "context"

func (db *DbT) OldestFromQueue(ctx context.Context) (order string, err error) {
	sql := "update queue set last_checked_at = default where order_num in (select order_num from queue order by last_checked_at limit 1) returning order_num;"
	resp := db.Pool.QueryRow(ctx, sql)
	err = resp.Scan(&order)
	return order, err
}

func (db *DbT) RemoveFromQueue(ctx context.Context) error {
	//sql := ""
	return nil
}
