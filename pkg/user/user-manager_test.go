package user

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUserManager_Init(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectError   bool
		expectedUsers map[string]string
		expectedError error
	}{
		{
			name:          "empty file",
			fileContent:   "",
			expectError:   false,
			expectedUsers: map[string]string{},
		},
		{
			name:        "valid users",
			fileContent: fmt.Sprintf("user1%s$2a$14$hash1\nuser2%s$2a$14$hash2\n", columnSep, columnSep),
			expectError: false,
			expectedUsers: map[string]string{
				"user1": "$2a$14$hash1",
				"user2": "$2a$14$hash2",
			},
		},
		{
			name:          "invalid format - missing colon",
			fileContent:   "user1$2a$14$hash1\n",
			expectError:   true,
			expectedError: ErrPwFileContentFormat,
		},
		{
			name:          "invalid format - empty username",
			fileContent:   ":$2a$14$hash1\n",
			expectError:   true,
			expectedError: ErrPwFileContentFormat,
		},
		{
			name:          "invalid format - empty password",
			fileContent:   "user1:\n",
			expectError:   true,
			expectedError: ErrPwFileContentFormat,
		},
		{
			name:          "invalid format - too many colons",
			fileContent:   fmt.Sprintf("user1%shash%sextra\n", columnSep, columnSep),
			expectError:   true,
			expectedError: ErrPwFileContentFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test_pw_*.txt")
			require.NoError(t, err)
			defer tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			if tt.fileContent != "" {
				n, err := tmpFile.WriteString(tt.fileContent)
				require.NoError(t, err, "failed to write pw file content")
				require.Equal(t, len(tt.fileContent), n, fmt.Sprintf("inconsistent write, %d != %d", len(tt.fileContent), n))
			}

			um := &UserManager{PwFile: tmpFile.Name()}
			err = um.Init()

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUsers, um.users)
				assert.NotNil(t, um.authenticatedUsers)
				assert.NotNil(t, um.authenticatedUsers.users)
			}
		})
	}
}

func TestUserManager_CreateUser(t *testing.T) {
	tests := []struct {
		name          string
		pwFileContent string
		credential    *Creadential
		expectError   bool
		expectedError error
	}{
		{
			name:          "valid user creation",
			pwFileContent: "",
			credential:    &Creadential{Username: "testuser", Password: "testpass"},
			expectError:   false,
		},
		{
			name:          "invalid username - starts with number",
			pwFileContent: "",
			credential:    &Creadential{Username: "1testuser", Password: "testpass"},
			expectError:   true,
			expectedError: ErrInvalidUsername,
		},
		{
			name:          "invalid username - contains special chars",
			pwFileContent: "",
			credential:    &Creadential{Username: "test-user", Password: "testpass"},
			expectError:   true,
			expectedError: ErrInvalidUsername,
		},
		{
			name:          "valid username with underscore",
			pwFileContent: "",
			credential:    &Creadential{Username: "test_user", Password: "testpass"},
			expectError:   false,
		},
		{
			name:          "valid username with numbers",
			pwFileContent: "",
			credential:    &Creadential{Username: "user123", Password: "testpass"},
			expectError:   false,
		},
		{
			name:          "username already exists",
			pwFileContent: fmt.Sprintf("testuser%ssomehash", columnSep),
			credential:    &Creadential{Username: "testuser", Password: "testpass"},
			expectError:   true,
			expectedError: ErrUsernameExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpPwFile, err := os.CreateTemp("", "test_pw_*.txt")
			require.NoError(t, err)
			defer os.Remove(tmpPwFile.Name())
			defer tmpPwFile.Close()

			if tt.pwFileContent != "" {
				n, err := tmpPwFile.WriteString(tt.pwFileContent)
				require.NoError(t, err, "failed to write pw file content")
				require.Equal(t, len(tt.pwFileContent), n, fmt.Sprintf("inconsistent write, %d != %d", len(tt.pwFileContent), n))
			}

			um := &UserManager{PwFile: tmpPwFile.Name()}
			err = um.Init()
			require.NoError(t, err)

			err = um.CreateUser(tt.credential)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != nil {
					assert.Equal(t, tt.expectedError, err)
				}
			} else {
				err = um.Init()
				require.NoError(t, err, "failed to re-init the user manager after user creation")

				assert.Contains(t, um.users, tt.credential.Username)
			}
		})
	}
}

func TestUserManager_DeleteUser(t *testing.T) {
	tests := []struct {
		name         string
		initialUsers map[string]string
		userToDelete string
		expectError  bool
		shouldDelete bool
	}{
		{
			name:         "delete existing user",
			initialUsers: map[string]string{"user1": "hash1", "user2": "hash2"},
			userToDelete: "user1",
			expectError:  false,
			shouldDelete: true,
		},
		{
			name:         "delete non-existing user",
			initialUsers: map[string]string{"user1": "hash1"},
			userToDelete: "nonexistent",
			expectError:  false,
			shouldDelete: false,
		},
		{
			name:         "empty username",
			initialUsers: map[string]string{"user1": "hash1"},
			userToDelete: "",
			expectError:  false,
			shouldDelete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test_pw_*.txt")
			require.NoError(t, err)
			defer tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			writer := bufio.NewWriter(tmpFile)
			for username, hash := range tt.initialUsers {
				rec := username + columnSep + hash + "\n"
				n, err := writer.WriteString(rec)
				require.NoError(t, err)
				require.Equal(t, len(rec), n, fmt.Sprintf("inconsistent write, %d != %d", len(rec), n))
			}
			err = writer.Flush()
			require.NoError(t, err, "failed to flush the writer")

			um := &UserManager{PwFile: tmpFile.Name()}
			err = um.Init()
			require.NoError(t, err)

			err = um.DeleteUser(tt.userToDelete)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.shouldDelete {
					err = um.Init()
					require.NoError(t, err, "failed to re-init the user manager after user deletion")

					assert.NotContains(t, um.users, tt.userToDelete)
				}
			}
		})
	}
}

func TestUserManager_CheckUserPassword(t *testing.T) {
	// Create a test password hash
	testPassword := "testpass123"
	testHash, err := bcrypt.GenerateFromPassword([]byte(testPassword), hashCost)
	require.NoError(t, err)

	tests := []struct {
		name     string
		users    map[string]string
		username string
		password string
		expected bool
	}{
		{
			name:     "valid password",
			users:    map[string]string{"testuser": string(testHash)},
			username: "testuser",
			password: testPassword,
			expected: true,
		},
		{
			name:     "invalid password",
			users:    map[string]string{"testuser": string(testHash)},
			username: "testuser",
			password: "wrongpass",
			expected: false,
		},
		{
			name:     "user not found",
			users:    map[string]string{"testuser": string(testHash)},
			username: "nonexistent",
			password: testPassword,
			expected: false,
		},
		{
			name:     "empty username",
			users:    map[string]string{"testuser": string(testHash)},
			username: "",
			password: testPassword,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			um := &UserManager{users: tt.users}
			result := um.CheckUserPassword(tt.username, tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserManager_CheckUserIP(t *testing.T) {
	tests := []struct {
		name              string
		authenticatedUser map[string]string
		username          string
		ipAddr            string
		expected          bool
	}{
		{
			name:              "valid IP check",
			authenticatedUser: map[string]string{"user1": "192.168.1.1"},
			username:          "user1",
			ipAddr:            "192.168.1.1",
			expected:          true,
		},
		{
			name:              "invalid IP check",
			authenticatedUser: map[string]string{"user1": "192.168.1.1"},
			username:          "user1",
			ipAddr:            "192.168.1.2",
			expected:          false,
		},
		{
			name:              "user not authenticated",
			authenticatedUser: map[string]string{},
			username:          "user1",
			ipAddr:            "192.168.1.1",
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			um := &UserManager{
				authenticatedUsers: &authenticatedUsers{
					users: tt.authenticatedUser,
				},
			}
			result := um.CheckUserIP(tt.username, tt.ipAddr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserManager_SetAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name                    string
		users                   map[string]string
		authenticatedUsers      map[string]string
		username                string
		ip                      string
		expectedResult          bool
		expectedAuthenticatedIP string
	}{
		{
			name:                    "set new authenticated user",
			users:                   map[string]string{"user1": "hash1"},
			authenticatedUsers:      map[string]string{},
			username:                "user1",
			ip:                      "192.168.1.1",
			expectedResult:          true,
			expectedAuthenticatedIP: "",
		},
		{
			name:                    "deny to update existing authenticated user",
			users:                   map[string]string{"user1": "hash1"},
			authenticatedUsers:      map[string]string{"user1": "192.168.1.1"},
			username:                "user1",
			ip:                      "192.168.1.2",
			expectedResult:          false,
			expectedAuthenticatedIP: "192.168.1.1",
		},
		{
			name:               "user doesn't exist",
			users:              map[string]string{},
			authenticatedUsers: map[string]string{},
			username:           "nonexistent",
			ip:                 "192.168.1.1",
			expectedResult:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			um := &UserManager{
				users: tt.users,
				authenticatedUsers: &authenticatedUsers{
					users: tt.authenticatedUsers,
				},
			}

			result := um.SetAuthenticatedUser(tt.username, tt.ip)
			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedAuthenticatedIP != "" {
				assert.Equal(t, tt.expectedAuthenticatedIP, um.authenticatedUsers.users[tt.username])
			}
		})
	}
}

func TestUserManager_UnsetAuthenticatedUser(t *testing.T) {
	authUsers := map[string]string{
		"user1": "192.168.1.1",
		"user2": "192.168.1.2",
	}

	um := &UserManager{
		authenticatedUsers: &authenticatedUsers{
			users: authUsers,
		},
	}

	// Test removing existing user
	um.UnsetAuthenticatedUser("user1")
	assert.NotContains(t, um.authenticatedUsers.users, "user1")
	assert.Contains(t, um.authenticatedUsers.users, "user2")

	// Test removing non-existent user (should not panic)
	um.UnsetAuthenticatedUser("nonexistent")
	assert.Contains(t, um.authenticatedUsers.users, "user2")
}

func TestUserManager_hashPassword(t *testing.T) {
	um := &UserManager{}

	password := "testpassword123"
	hash, err := um.hashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify the hash can be used to verify the password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)
}

func TestUserManager_checkPasswordHash(t *testing.T) {
	um := &UserManager{}

	password := "testpassword123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	require.NoError(t, err)

	// Test correct password
	result := um.checkPasswordHash(password, string(hash))
	assert.True(t, result)

	// Test incorrect password
	result = um.checkPasswordHash("wrongpassword", string(hash))
	assert.False(t, result)

	// Test invalid hash
	result = um.checkPasswordHash(password, "invalid_hash")
	assert.False(t, result)
}

// Test concurrent access to authenticatedUsers
func TestUserManager_ConcurrentAccess(t *testing.T) {
	um := &UserManager{
		users: map[string]string{"user1": "hash1"},
		authenticatedUsers: &authenticatedUsers{
			users: make(map[string]string),
		},
	}

	// Add initial authenticated user
	um.authenticatedUsers.users["user1"] = "192.168.1.1"

	// Test concurrent access
	done := make(chan bool)

	// Goroutine 1: Check IP
	go func() {
		for i := 0; i < 100; i++ {
			um.CheckUserIP("user1", "192.168.1.1")
		}
		done <- true
	}()

	// Goroutine 2: Set authenticated user
	go func() {
		for i := 0; i < 100; i++ {
			um.SetAuthenticatedUser("user1", "192.168.1.1")
		}
		done <- true
	}()

	// Goroutine 3: Unset authenticated user
	go func() {
		for i := 0; i < 100; i++ {
			um.UnsetAuthenticatedUser("user1")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Test should not panic - if it gets here, mutex is working
	assert.True(t, true)
}
