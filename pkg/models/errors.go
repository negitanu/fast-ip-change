package models

import "errors"

var (
	ErrInvalidProfileName = errors.New("プロファイル名が無効です")
	ErrInvalidIPAddress   = errors.New("IPアドレスが無効です")
	ErrInvalidSubnetMask  = errors.New("サブネットマスクが無効です")
	ErrInvalidNICName     = errors.New("NIC名が無効です")
)
