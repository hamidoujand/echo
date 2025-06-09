package app

import "fmt"

func systemErrorMessage(format string, args ...any) message {
	return message{
		Name: "system",
		Text: fmt.Appendf(nil, format, args...),
	}
}
