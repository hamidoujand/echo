package app

import "fmt"

func formatMessage(name, msg string) string {
	return fmt.Sprintf("%s: %s", name, msg)
}
