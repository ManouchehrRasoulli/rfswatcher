package user

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type authenticatedUsers struct {
	users map[string]string
	mutex sync.Mutex
}

type UserManager struct {
	PwFile             string
	users              map[string]string   // keys: username / values: password hash
	authenticatedUsers *authenticatedUsers // keys: username / values: ip address
}

type Creadential struct {
	Username string
	Password string
}

var (
	ErrInvalidUsername     = errors.New("username is invalid. it can contains letters, numbers and underscores but should starts with a letter")
	ErrUsernameExists      = errors.New("username exists")
	ErrPwFileContentFormat = errors.New("something is wrong with the password file content format")
)

const (
	columnSep  = ":"
	hashCost   = 14
	pwFileMode = 0600
)

func (m *UserManager) Init() error {
	m.users = make(map[string]string)
	m.authenticatedUsers = &authenticatedUsers{users: make(map[string]string)}

	f, err := os.OpenFile(m.PwFile, os.O_CREATE|os.O_RDONLY, pwFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		userFields := strings.Split(line, columnSep)
		if len(userFields) != 2 || userFields[0] == "" || userFields[1] == "" {
			subErr := fmt.Errorf("(len: %d, fields: %v)", len(userFields), userFields)
			return errors.Join(ErrPwFileContentFormat, subErr)
		}
		m.users[userFields[0]] = userFields[1]
	}

	err = scanner.Err()
	if err != nil {
		return err
	}

	return nil
}

func (m *UserManager) CreateUser(cred *Creadential) error {
	var username string
	var password string
	usernameRegex := regexp.MustCompile(`^[a-zA-Z]\w*$`)
	if cred != nil {
		if usernameRegex.MatchString(cred.Username) {
			if _, ok := m.users[cred.Username]; ok {
				return ErrUsernameExists
			} else {
				username = cred.Username
			}
		} else {
			return ErrInvalidUsername
		}

		password = cred.Password
	} else {
		for {
			fmt.Print("Enter username: ")
			fmt.Scanln(&username)
			if usernameRegex.MatchString(username) {
				if _, ok := m.users[username]; ok {
					fmt.Println(ErrUsernameExists)
				} else {
					break
				}
			} else {
				fmt.Println(ErrInvalidUsername)
			}
		}

		fmt.Printf("Password for %s: ", username)
		fmt.Scanln(&password)

		var passwordConfirm string
		fmt.Printf("Confirm password for %s: ", username)
		fmt.Scanln(&passwordConfirm)
		if password != passwordConfirm {
			return errors.New("ERROR: Password does not match.")
		}
	}

	hashPass, err := m.hashPassword(password)
	if err != nil {
		return err
	}

	userRecord := username + columnSep + hashPass + "\n"
	f, err := os.OpenFile(m.PwFile, os.O_APPEND|os.O_WRONLY, pwFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(userRecord)
	if err != nil {
		return err
	}

	return nil
}

func (m *UserManager) DeleteUser(username string) error {
	if username == "" {
		fmt.Print("Enter username to DELETE: ")
		fmt.Scanln(&username)
	}

	if _, ok := m.users[username]; ok {
		delete(m.users, username)

		originalPw, err := os.OpenFile(m.PwFile, os.O_RDONLY, pwFileMode)
		if err != nil {
			return err
		}
		defer originalPw.Close()

		tmpPw, err := os.CreateTemp(filepath.Dir(m.PwFile), "pwfile_*.tmp")
		if err != nil {
			return err
		}
		defer tmpPw.Close()
		defer os.Remove(tmpPw.Name())

		scanner := bufio.NewScanner(originalPw)
		writer := bufio.NewWriter(tmpPw)

		for scanner.Scan() {
			line := scanner.Text()
			userFields := strings.Split(line, columnSep)

			if len(userFields) != 2 || userFields[0] == "" || userFields[1] == "" || userFields[0] == username {
				continue
			}

			_, err = writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		if err := writer.Flush(); err != nil {
			return nil
		}

		if err := os.Rename(tmpPw.Name(), m.PwFile); err != nil {
			return err
		}

		if err := os.Chmod(m.PwFile, pwFileMode); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (m *UserManager) CheckUserPassword(username, password string) bool {
	if passwordHash, ok := m.users[username]; ok {
		return m.checkPasswordHash(password, passwordHash)
	}

	return false
}

func (m *UserManager) CheckUserIP(username, ipAddr string) bool {
	m.authenticatedUsers.mutex.Lock()
	defer m.authenticatedUsers.mutex.Unlock()
	if userIP, ok := m.authenticatedUsers.users[username]; ok {
		return ipAddr == userIP
	} else {
		return false
	}
}

// Set username/ip to the authenticated users map.
// returns false if the user not exists or it already exists in authenticated users map.
func (m *UserManager) SetAuthenticatedUser(username, ip string) (ok bool) {
	if _, ok := m.users[username]; ok {
		m.authenticatedUsers.mutex.Lock()
		defer m.authenticatedUsers.mutex.Unlock()
		if _, ok := m.authenticatedUsers.users[username]; !ok {
			m.authenticatedUsers.users[username] = ip
			return true
		}

		return false
	}

	return false
}

// Removes username from authenticated users map.
func (m *UserManager) UnsetAuthenticatedUser(username string) {
	m.authenticatedUsers.mutex.Lock()
	defer m.authenticatedUsers.mutex.Unlock()
	delete(m.authenticatedUsers.users, username)
}

func (m *UserManager) hashPassword(password string) (string, error) {
	byts, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)

	return string(byts), err
}

func (m *UserManager) checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
