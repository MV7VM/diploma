package entities

import "time"

const (
	OrderStatusInvalid string = "INVALID"
)

type CtxKeyString string

type UserAuth struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Order struct {
	Number     string    `json:"number" db:"order_number"`
	Status     string    `json:"status" db:"status"`
	Accrual    *int      `json:"accrual,omitempty" db:"accrual"`
	UploadedAt time.Time `json:"uploaded_at" db:"upload_time"`
}

type Withdraw struct {
	OrderNumber string    `json:"order" db:"order_number"`
	Sum         float64   `json:"sum" db:"sum"`
	UploadedAt  time.Time `json:"processed_at" db:"upload_time"`
}

type Accrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
