package errs

import (
	"runtime"
	"strconv"
)

type Error struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	FuncName string `json:"-"`
	Filename string `json:"-"`
}

func New(code int, err error) *Error {
	//skip 1 call stack and capture PC from where this error was created
	pc, filename, line, _ := runtime.Caller(1)
	f := filename + ":" + strconv.Itoa(line)
	return &Error{
		Code:     code,
		Message:  err.Error(),
		FuncName: runtime.FuncForPC(pc).Name(),
		Filename: f,
	}
}

func (e *Error) Error() string {
	return e.Message
}
