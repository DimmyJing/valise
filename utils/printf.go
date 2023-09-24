package utils

import "fmt"

func EmulatePrintf(format string, args ...any) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
}

func EmulateErrorf(format string, args ...any) {
	if false {
		//nolint:goerr113
		_ = fmt.Errorf(format, args...)
	}
}
