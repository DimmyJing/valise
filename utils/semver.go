package utils

import (
	"cmp"
	"errors"
	"strconv"
	"strings"
)

var errInvalidSemVer = errors.New("invalid semver")

// CompareSemVer returns
//
//	-1 if a is less than b,
//	 0 if a equals b,
//	+1 if a is greater than b.
func CompareSemVer(a string, b string) (int, error) {
	aSplit := strings.Split(a, ".")
	bSplit := strings.Split(b, ".")

	if len(aSplit) != 3 || len(bSplit) != 3 {
		return 0, errInvalidSemVer
	}

	for semVerIdx := 0; semVerIdx < 3; semVerIdx++ {
		first, err := strconv.Atoi(aSplit[semVerIdx])
		if err != nil {
			return 0, errInvalidSemVer
		}

		second, err := strconv.Atoi(bSplit[semVerIdx])
		if err != nil {
			return 0, errInvalidSemVer
		}

		res := cmp.Compare(first, second)
		if res != 0 {
			return res, nil
		}
	}

	return 0, nil
}
