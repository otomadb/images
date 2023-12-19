package imgproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"

	"os"
)

func SignPath(path string) (string, error) {
	var key = os.Getenv("IMGPROXY_KEY")
	if key == "" {
		panic("IMGPROXY_KEY is not set")
	}

	var salt = os.Getenv("IMGPROXY_SALT")
	if salt == "" {
		panic("IMGPROXY_SALT is not set")
	}

	var baseUrl = os.Getenv("IMGPROXY_URL")
	if baseUrl == "" {
		panic("IMGPROXY_URL is not set")
	}

	var keyBin, saltBin []byte
	var err error

	if keyBin, err = hex.DecodeString(key); err != nil {
		log.Fatal(err)
		return "", err
	}

	if saltBin, err = hex.DecodeString(salt); err != nil {
		log.Fatal(err)
		return "", err
	}

	mac := hmac.New(sha256.New, keyBin)
	mac.Write(saltBin)
	mac.Write([]byte(path))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return baseUrl + "/" + signature + path, nil
}
