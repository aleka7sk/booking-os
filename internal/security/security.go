package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"
)

const passwordIterations = 120_000

func RandomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashPassword(password string) (saltEncoded, hashEncoded string, err error) {
	salt := make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return "", "", err
	}
	hash := pbkdf2SHA256([]byte(password), salt, passwordIterations, 32)
	return base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash), nil
}

func VerifyPassword(password, saltEncoded, expectedEncoded string) bool {
	salt, err := base64.RawStdEncoding.DecodeString(saltEncoded)
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(expectedEncoded)
	if err != nil {
		return false
	}
	actual := pbkdf2SHA256([]byte(password), salt, passwordIterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	hLen := 32
	blocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, blocks*hLen)
	var counter [4]byte
	for block := 1; block <= blocks; block++ {
		binary.BigEndian.PutUint32(counter[:], uint32(block))
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write(counter[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func SignSession(secret, userID string, expiresAt time.Time) string {
	payload := userID + "|" + strconv.FormatInt(expiresAt.Unix(), 10)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig
}

func VerifySession(secret, token string, now time.Time) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid session token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, expected) {
		return "", errors.New("invalid session signature")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid session payload")
	}
	fields := strings.Split(string(decoded), "|")
	if len(fields) != 2 {
		return "", errors.New("invalid session payload")
	}
	expiresUnix, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || now.After(time.Unix(expiresUnix, 0)) {
		return "", errors.New("session expired")
	}
	return fields[0], nil
}
