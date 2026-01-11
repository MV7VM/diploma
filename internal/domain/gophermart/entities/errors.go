package entities

const ErrLoginAlreadyInUse MyError = "login already in use"
const ErrWrongCredentials MyError = "incorrect login or password"

type MyError string

func (e MyError) Error() string {
	return string(e)
}
