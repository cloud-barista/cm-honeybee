package ssh

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	_ssh "golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
)

type SSH struct {
	Options Options
	// command string
}

type Options struct {
	SSHAddress               string
	SSHPort                  int
	SSHUsername              string
	SSHPassword              string
	IdentityFilePath         string
	IdentityFilePathProvided bool
	session                  *_ssh.Session
	client                   *_ssh.Client
}

func DefaultSSHOptions() Options {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("failed to determine user home directory: %v\n", err)
	}
	options := Options{
		SSHPort:                  22,
		SSHUsername:              defaultUsername(),
		SSHPassword:              "",
		IdentityFilePath:         filepath.Join(homeDir, ".ssh", "id_rsa"),
		IdentityFilePathProvided: false,
	}
	return options
}

func (o *SSH) NewClientConn(connectionInfo model.ConnectionInfo) error {
	addr := fmt.Sprintf("%s:%d", connectionInfo.IPAddress, connectionInfo.SSHPort)

	sshConfig := &_ssh.ClientConfig{
		User:            connectionInfo.User,
		Auth:            o.getAuthMethods(connectionInfo),
		HostKeyCallback: _ssh.InsecureIgnoreHostKey(),
	}

	client, err := _ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return err
	}
	fmt.Println("SSH Connection Success. ")

	o.Options.client = client
	defer o.Close()

	session, err := client.NewSession()
	if err != nil {
		// client.Close()
		return fmt.Errorf("failed to create session: %s", err)
	}
	o.Options.session = session

	session.Stdin = os.Stdin
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	_ = o.RunCmd("cat $HOME/.ssh/id_rsa")
	_ = o.RunCmd("cat $HOME/.ssh/id_ed25519")

	return nil
}

func (o *SSH) getAuthMethods(connectionInfo model.ConnectionInfo) []_ssh.AuthMethod {
	var methods []_ssh.AuthMethod

	methods = o.tryPrivateKey(methods)

	methods = append(methods, _ssh.PasswordCallback(func() (secret string, err error) {
		return connectionInfo.Password, nil
	}))

	return methods
}

func (o *SSH) tryPrivateKey(methods []_ssh.AuthMethod) []_ssh.AuthMethod {

	if !o.Options.IdentityFilePathProvided {
		if _, err := os.Stat(o.Options.IdentityFilePath); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("No ssh key at the default location %q found, skipping RSA authentication.\n", o.Options.IdentityFilePath)
			return methods
		}
	}

	callback := _ssh.PublicKeysCallback(func() (signers []_ssh.Signer, err error) {
		key, err := os.ReadFile(o.Options.IdentityFilePath)
		if err != nil {
			return nil, err
		}

		signer, err := _ssh.ParsePrivateKey(key)
		var passphraseMissingError *_ssh.PassphraseMissingError
		isPassErr := errors.As(err, &passphraseMissingError)
		if isPassErr {
			signer, err = o.parsePrivateKeyWithPassphrase(key)
		}
		if err != nil {
			return nil, err
		}

		return []_ssh.Signer{signer}, nil
	})

	return append(methods, callback)
}

func (o *SSH) parsePrivateKeyWithPassphrase(key []byte) (_ssh.Signer, error) {
	//password, err := readPassword(fmt.Sprintf("Key %s requires a password: ", o.Options.IdentityFilePath))
	fmt.Println()
	//if err != nil {
	//	return nil, err
	//}

	return _ssh.ParsePrivateKeyWithPassphrase(key, []byte{})
}

// RunCmd to SSH Server
func (o *SSH) RunCmd(cmd string) error {

	if cmd != "" {
		if err := o.Options.session.Run(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (o *SSH) Close() {
	if o.Options.session != nil {
		_ = o.Options.session.Close()
	}
	_ = o.Options.client.Close()
}

func defaultUsername() string {
	vars := []string{
		"USER",     // linux
		"USERNAME", // linux, windows
		"LOGNAME",  // linux
	}
	for _, env := range vars {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

//import "golang.org/x/term"
//func readPassword(reason string) ([]byte, error) {
//	fmt.Print(reason)
//	return term.ReadPassword(int(os.Stdin.Fd()))
//}
