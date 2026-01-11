package entities

const (
	ErrAlreadyInUse     MyError = "already in use"
	ErrWrongCredentials MyError = "incorrect login or password"
	ErrPermissionDenied MyError = "permission denied"
)

type MyError string

func (e MyError) Error() string {
	return string(e)
}
