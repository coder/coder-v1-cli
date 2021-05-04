package wsnet

import (
	"crypto/sha256"
	"errors"
	"strings"
)

// TURNCredentials returns a username and password pair
// for a Coder token.
func TURNCredentials(token string) (username, password string, err error) {
	str := strings.SplitN(token, "-", 2)
	if len(str) != 2 {
		err = errors.New("invalid token format")
		return
	}
	username = str[0]
	hash := sha256.Sum256([]byte(str[1]))
	password = string(hash[:])
	return
}
