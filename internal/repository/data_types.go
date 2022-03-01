package repository

import (
	"errors"
	"time"
)

// ошибки обращения к репозиторию
var (
	ErrLoginBusy                         = errors.New("login is busy")
	ErrUnknownLogin                      = errors.New("unknown login")
	ErrDuplicateOrderNumber              = errors.New("order number already loaded")
	ErrDuplicateOrderNumberByAnotherUser = errors.New("order number already loaded by another user")
	ErrNotEnoughFunds                    = errors.New("not enough funds in account")
	ErrOrderAlreadyExists                = errors.New("order already exists")
	ErrEmptyQueue                        = errors.New("queue is empty")
)

// статусы начисления баллов заказам
var (
	AccrualNew        = "NEW"
	AccrualInvalid    = "INVALID"
	AccrualProcessing = "PROCESSING"
	AccrualProcessed  = "PROCESSED"
)

type RegisterNewUser struct {
	Login   string
	PwdHash string
	PwdSalt string
	JWTSalt string
}

type LoginUser struct {
	UserID  int
	PwdHash string
	PwdSalt string
}

type OrderList []orderItem
type orderItem struct {
	Number       string    `json:"number"`
	Status       string    `json:"status"`
	Accrual      float64   `json:"accrual,omitempty"`
	UploadedAt   string    `json:"uploaded_at"`
	UploadedAtGo time.Time `json:"-"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawalsList []withdrawalItem
type withdrawalItem struct {
	Order         string    `json:"order"`
	Sum           float64   `json:"sum"`
	ProcessedAt   string    `json:"processed_at"`
	ProcessedAtGo time.Time `json:"-"`
}
