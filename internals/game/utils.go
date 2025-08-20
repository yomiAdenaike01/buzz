package game

import (
	"crypto/rand"
	"math/big"
	"strings"
)

func generateShortCode() (string, error) {
	maxLen := 4
	var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	l := len(alphabet)
	var builder strings.Builder
	builder.Grow(maxLen)
	for range maxLen {
		i, err := rand.Int(rand.Reader, big.NewInt(int64(l)))
		if err != nil {
			return "", err
		}
		builder.WriteRune(alphabet[i.Int64()])
	}
	return builder.String(), nil
}
