package env

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DimmyJing/valise/log"
)

func encrypt(key []byte, val string) string {
	aesIV := make([]byte, aes.BlockSize)

	_, err := rand.Read(aesIV)
	if err != nil {
		log.Panic(err)
	}

	rawValue := []byte(val)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Panic(err)
	}

	cfb := cipher.NewCFBEncrypter(block, aesIV)
	cipherText := make([]byte, len(rawValue))
	cfb.XORKeyStream(cipherText, rawValue)
	//nolint:makezero
	cipherText = append(aesIV, cipherText...)
	encryptedValue := base64.StdEncoding.EncodeToString(cipherText)

	return "enc:" + encryptedValue
}

func decrypt(key []byte, val string) string {
	if !strings.HasPrefix(val, "enc:") {
		return val
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Panic(err)
	}

	rawValue, err := base64.StdEncoding.DecodeString(val[4:])
	if err != nil {
		log.Panic(err)
	}

	iv := rawValue[:aes.BlockSize]
	cipherText := rawValue[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)

	return string(plainText)
}

const minDirLen = 5

func findFile(filename string) []string {
	candidates := []string{}
	_, b, _, _ := runtime.Caller(0)
	dir := filepath.Dir(b)

	for {
		if len(dir) < minDirLen {
			break
		} else if _, err := os.Stat(filepath.Join(dir, filename)); !errors.Is(err, os.ErrNotExist) {
			candidates = append(candidates, filepath.Join(dir, filename))
		}

		dir = filepath.Dir(dir)
	}

	return candidates
}

var errKeyNotFound = errors.New("key not found")

func getKey() ([]byte, error) {
	if key, found := os.LookupEnv("KEY"); found {
		res, err := hex.DecodeString(key)
		if err != nil {
			return res, fmt.Errorf("failed to decode key: %w", err)
		}

		return res, nil
	}

	for _, candidate := range findFile(".key") {
		dat, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		res, err := hex.DecodeString(string(dat))
		if err != nil {
			return res, fmt.Errorf("failed to decode key: %w", err)
		}

		return res, nil
	}

	return []byte{}, errKeyNotFound
}

var errInvalidKey = errors.New(
	"failed to get env decryption key, there must either be a .key file in the root of the project or a KEY env var",
)

func InitEnv(envJSON []byte) error {
	secretKey, err := getKey()
	if err != nil {
		return errInvalidKey
	}

	envVars := make(map[string]string)

	err = json.Unmarshal(envJSON, &envVars)
	if err != nil {
		return fmt.Errorf("error umarshalling env json: %w", err)
	}

	for key, value := range envVars {
		if _, found := os.LookupEnv(key); !found {
			err = os.Setenv(key, decrypt(secretKey, value))
			if err != nil {
				return fmt.Errorf("error setting env: %w", err)
			}
		}
	}

	return nil
}

var errEnvNotFound = errors.New("environmental variable not found")

func Get(key string) string {
	if env, found := os.LookupEnv(key); found {
		return env
	}

	log.Panic(fmt.Errorf("%w: %s", errEnvNotFound, key))

	return ""
}

func GetDefault(key string, defaultValue string) string {
	if env, found := os.LookupEnv(key); found {
		return env
	}

	return defaultValue
}
