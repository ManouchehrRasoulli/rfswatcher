package pkg

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ManouchehrRasoulli/rfswatcher/pkg/client"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/server"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/user"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/watcher"
	"github.com/stretchr/testify/require"
)

func checkOpenSSL() error {
	cmd := exec.Command("openssl", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("OpenSSL not available %w", err)
	}

	return nil
}

func genTlsFiles() (key string, crt string, err error) {
	fKey, err := os.CreateTemp("", "*.key")
	if err != nil {
		return "", "", err
	}
	defer fKey.Close()

	fCrt, err := os.CreateTemp("", "*.crt")
	if err != nil {
		return "", "", err
	}
	defer fCrt.Close()

	genKeyCmd := exec.Command("openssl", "genrsa", "-out", fKey.Name(), "2048")
	if out, err := genKeyCmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w, output: %s", err, out)
	}

	genCrtCmd := exec.Command("openssl", "req", "-new", "-x509", "-key", fKey.Name(),
		"-out", fCrt.Name(), "-days", "1", "-subj", "/CN=localhost")
	if output, err := genCrtCmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("failed to generate certificate: %w, output: %s", err, output)
	}

	return fKey.Name(), fCrt.Name(), nil
}

func genPwFile() (username string, password string, um *user.UserManager, err error) {
	fPwFile, err := os.CreateTemp("", "pw_*.txt")
	if err != nil {
		return "", "", nil, err
	}
	defer fPwFile.Close()

	um = &user.UserManager{PwFile: fPwFile.Name()}
	username = "user"
	password = "user"

	if err = um.CreateUser(&user.Creadential{Username: username, Password: password}); err != nil {
		return "", "", nil, err
	}

	if err = um.Init(); err != nil {
		return "", "", nil, err
	}

	return username, password, um, nil
}

func TestIntegration(t *testing.T) {
	t.Log("Start integration test ...")
	lg := log.New(os.Stdout, "integration --> ", 1|4)

	fileHandler, err := filehandler.NewHandler(".", lg)
	require.NoError(t, err, "internal handler !")

	address := "localhost:9801"
	s := server.NewServer(address, ".", nil, nil, lg, fileHandler)
	w, err := watcher.NewWatcher(".", watcher.WithCallbackFunction(fileHandler.EventHook), watcher.WithCallbackFunction(s.EventHook))
	require.NoError(t, err, "new watcher error !")
	defer w.Close()

	go func() {
		err := s.Run()
		require.NoError(t, err, "server error !")
	}()

	time.Sleep(time.Second)

	c := client.NewClient(address, "", "", nil, lg, fileHandler)
	go func() {
		err := c.Run()
		require.NoError(t, err, "client error !")
	}()

	exit := make(chan struct{})
	_ = time.AfterFunc(time.Second*2, func() {
		defer close(exit)
		err = c.Exit()
		require.NoError(t, err, "client exit !!")

		err = s.Exit()
		require.NoError(t, err, "server exit !!")
	})

	<-exit
	t.Log("Integration test done.")
}

func TestIntegrationWithTLS(t *testing.T) {
	if err := checkOpenSSL(); err != nil {
		t.Skip("Skipping TLS test: ", err)
	}

	t.Log("Start integration test with TLS ...")
	lg := log.New(os.Stdout, "integration tls --> ", 1|4)

	fileHandler, err := filehandler.NewHandler(".", lg)
	require.NoError(t, err, "failed to init file handler")

	key, crt, err := genTlsFiles()
	require.NoError(t, err, "failed to generate TLS files")
	defer os.Remove(key)
	defer os.Remove(crt)

	address := "localhost:9802"
	tlsCfg := &server.ServerTLS{Key: key, Cert: crt}
	s := server.NewServer(address, ".", tlsCfg, nil, lg, fileHandler)
	w, err := watcher.NewWatcher(".", watcher.WithCallbackFunction(fileHandler.EventHook), watcher.WithCallbackFunction(s.EventHook))
	require.NoError(t, err, "failed to init watcher")
	defer w.Close()

	go func() {
		err := s.Run()
		require.NoError(t, err, "failed to run server")
	}()

	time.Sleep(time.Second)

	cTlsCfg := &tls.Config{InsecureSkipVerify: true}
	c := client.NewClient(address, "", "", cTlsCfg, lg, fileHandler)
	go func() {
		err := c.Run()
		require.NoError(t, err, "failed to run client")
	}()

	exit := make(chan struct{})
	_ = time.AfterFunc(time.Second*2, func() {
		defer close(exit)
		err = c.Exit()
		require.NoError(t, err, "client exit !!")

		err = s.Exit()
		require.NoError(t, err, "server exit !!")
	})

	<-exit
	t.Log("Integration test with TLS done.")
}

func TestIntegrationWithPW(t *testing.T) {
	t.Log("Start integration test with password file ...")
	lg := log.New(os.Stdout, "integration pw --> ", 1|4)

	fileHandler, err := filehandler.NewHandler(".", lg)
	require.NoError(t, err, "failed to init file handler")

	username, password, um, err := genPwFile()
	require.NoError(t, err, "failed to generate password file")
	defer os.Remove(um.PwFile)

	address := "localhost:9803"
	s := server.NewServer(address, ".", nil, um, lg, fileHandler)
	w, err := watcher.NewWatcher(".", watcher.WithCallbackFunction(fileHandler.EventHook), watcher.WithCallbackFunction(s.EventHook))
	require.NoError(t, err, "failed to init watcher")
	defer w.Close()

	go func() {
		err := s.Run()
		require.NoError(t, err, "failed to run server")
	}()

	time.Sleep(time.Second)

	c := client.NewClient(address, username, password, nil, lg, fileHandler)
	go func() {
		err := c.Run()
		require.NoError(t, err, "failed to run client")
	}()

	exit := make(chan struct{})
	_ = time.AfterFunc(time.Second*2, func() {
		defer close(exit)
		err = c.Exit()
		require.NoError(t, err, "client exit !!")

		err = s.Exit()
		require.NoError(t, err, "server exit !!")
	})

	<-exit
	t.Log("Integration test with password file done.")
}

func TestIntegrationWithTLSAndPW(t *testing.T) {
	if err := checkOpenSSL(); err != nil {
		t.Skip("Skipping TLS test: ", err)
	}

	t.Log("Start integration test with TLS and password file ...")
	lg := log.New(os.Stdout, "integration tls/pw --> ", 1|4)

	fileHandler, err := filehandler.NewHandler(".", lg)
	require.NoError(t, err, "failed to init file handler")

	key, crt, err := genTlsFiles()
	require.NoError(t, err, "failed to generate TLS files")
	defer os.Remove(key)
	defer os.Remove(crt)

	username, password, um, err := genPwFile()
	require.NoError(t, err, "failed to generate password file")
	defer os.Remove(um.PwFile)

	address := "localhost:9804"
	tlsCfg := &server.ServerTLS{Key: key, Cert: crt}
	s := server.NewServer(address, ".", tlsCfg, um, lg, fileHandler)
	w, err := watcher.NewWatcher(".", watcher.WithCallbackFunction(fileHandler.EventHook), watcher.WithCallbackFunction(s.EventHook))
	require.NoError(t, err, "failed to init watcher")
	defer w.Close()

	go func() {
		err := s.Run()
		require.NoError(t, err, "failed to run server")
	}()

	time.Sleep(time.Second)

	cTlsCfg := &tls.Config{InsecureSkipVerify: true}
	c := client.NewClient(address, username, password, cTlsCfg, lg, fileHandler)
	go func() {
		err := c.Run()
		require.NoError(t, err, "failed to run client")
	}()

	exit := make(chan struct{})
	_ = time.AfterFunc(time.Second*2, func() {
		defer close(exit)
		err = c.Exit()
		require.NoError(t, err, "client exit !!")

		err = s.Exit()
		require.NoError(t, err, "server exit !!")
	})

	<-exit
	t.Log("Integration test with TLS and password file done.")
}
