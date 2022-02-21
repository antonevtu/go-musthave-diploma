package repository

import "errors"

var ErrLoginBusy = errors.New("login is busy")
var ErrInvalidLoginPassword = errors.New("invalid login/password pair")

var ErrDuplicateOrderNumber = errors.New("order number already loaded")
var ErrDuplicateOrderNumberByAnotherUser = errors.New("order number already loaded by another user")

var ErrNotEnoughFunds = errors.New("not enough funds in account")

type OrdersList []orderItem
type orderItem struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadedAt string `json:"uploaded_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawalsList []withdrawalItem
type withdrawalItem struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
