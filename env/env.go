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

func GetEnvFromJSON(envJSON []byte, key []byte, envKey string) (string, error) {
	envVars := make(map[string]string)

	err := json.Unmarshal(envJSON, &envVars)
	if err != nil {
		return "", fmt.Errorf("error umarshalling env json: %w", err)
	}

	if val, found := envVars[envKey]; found {
		return decrypt(key, val), nil
	}

	return "", nil
}

func SetEnvFromJSON(envJSON []byte, key []byte, envKey string, envVal string, enc bool) ([]byte, error) {
	envVars := make(map[string]string)

	err := json.Unmarshal(envJSON, &envVars)
	if err != nil {
		return nil, fmt.Errorf("error umarshalling env json: %w", err)
	}

	if enc {
		envVars[envKey] = encrypt(key, envVal)
	} else {
		envVars[envKey] = envVal
	}

	newEnvJSON, err := json.MarshalIndent(envVars, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling env json: %w", err)
	}

	return newEnvJSON, nil
}

const minDirLen = 5

func findFile(filename string) []string {
	candidates := []string{}
	//nolint:mnd
	_, b, _, _ := runtime.Caller(2)
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

func GetKey() ([]byte, error) {
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

func Init(envJSON []byte, secretKey []byte) error {
	envVars := make(map[string]string)

	err := json.Unmarshal(envJSON, &envVars)
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
