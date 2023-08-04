package functions

import (
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

var usersPasswords = map[string][]byte{}

type AuthFile struct {
	Users map[string]string `yaml:"basic_auth_users"`
}

func VerifyUserPass(username, password string) bool {
	wantPass, hasUser := usersPasswords[username]
	if !hasUser {
		return false
	}
	if cmperr := bcrypt.CompareHashAndPassword(wantPass, []byte(password)); cmperr == nil {
		return true
	}
	return false
}

func ReadAuthFile(fname string) error {
	yfile, err := os.ReadFile(fname)
	if err != nil {
		log.Fatal(err)
	}

	users := AuthFile{}
	err = yaml.Unmarshal(yfile, &users)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range users.Users {
		usersPasswords[k] = []byte(v)
	}
	return nil
}
