package entities

import "time"

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
	OrderNumber string `json:"order" db:"order_number"`
	Sum         int    `json:"sum" db:"sum"`
}
