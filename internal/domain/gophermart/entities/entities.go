package entities

type CtxKeyString string

type UserAuth struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}
