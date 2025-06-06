package app

import "fmt"

func formatMessage(name string, msg []byte) []byte {
	return fmt.Appendf(nil, "%s: %s", name, msg)
}

func systemErrorMessage(format string, args ...any) message {
	return message{
		Name: "system",
		Text: fmt.Appendf(nil, format, args...),
	}
}
