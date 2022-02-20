package repository

import "errors"

var ErrLoginBusy = errors.New("login is busy")
var ErrInvalidLoginPassword = errors.New("invalid login/password pair")
