package utils

import gonanoid "github.com/matoous/go-nanoid/v2"

func NanoID() string {
	return gonanoid.Must()
}

const (
	nanoIDLen = 21
	alphabet  = "_0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func NanoIDAlpha() string {
	return gonanoid.MustGenerate(alphabet, nanoIDLen)
}
