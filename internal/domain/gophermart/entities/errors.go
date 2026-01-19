package entities

const (
	ErrAlreadyInUse     MyError = "already in use"
	ErrWrongCredentials MyError = "incorrect login or password"
	ErrPermissionDenied MyError = "permission denied"
	ErrEmptyBalance     MyError = "balance is less then withdraw"
)

type MyError string

func (e MyError) Error() string {
	return string(e)
}
