package utils

import "fmt"

func EmulatePrintf(format string, args ...any) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
}
