package ssh

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"

	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	ConnectionInfo *model.ConnectionInfo
	Options        Options
}

type Response struct {
	StatusCode string              `json:"status_code"`
	Type       string              `json:"type"`
	Data       model.BenchmarkData `json:"data"`
}

type Options struct {
	SSHAddress               string
	SSHPort                  int
	SSHUsername              string
	SSHPassword              string
	IdentityFilePath         string
	IdentityFilePathProvided bool
	session                  *ssh.Session
	client                   *ssh.Client
}

//go:embed sourceFiles/*
var sourceFiles embed.FS

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

	sshConfig := &ssh.ClientConfig{
		User:            connectionInfo.User,
		Auth:            o.getAuthMethods(connectionInfo),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return err
	}
	logger.Println(logger.INFO, false, "SSH Connection Success. (IP: "+connectionInfo.IPAddress+
		" Port: "+strconv.Itoa(connectionInfo.SSHPort)+", User: "+connectionInfo.User+")")

	o.ConnectionInfo = &connectionInfo
	o.Options.client = client

	return nil
}

func (o *SSH) RunBenchmark(connectionInfo model.ConnectionInfo) ([]model.Benchmark, error) {
	err := o.NewClientConn(connectionInfo)
	if err != nil {
		return nil, err
	}
	defer func() {
		o.Close()
	}()

	// SFTP Client 설정
	client, err := sftp.NewClient(o.Options.client)
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to SFTP Connect: "+err.Error())
		return nil, err
	}
	defer func() {
		_ = client.Close()
	}()

	dstPath := "/tmp/"

	files := []string{"sourceFiles/milkyway", "sourceFiles/milkyway.sh"}

	for _, file := range files {
		fileContents, err := sourceFiles.ReadFile(file)
		if err != nil {
			logger.Println(logger.ERROR, true, "SSH: Failed to read source file: "+err.Error())
			return nil, err
		}

		dstFilePath := filepath.Join(dstPath, strings.Split(file, "/")[1])
		dstFile, err := client.Create(dstFilePath)
		if err != nil {
			logger.Println(logger.ERROR, true, "SSH: Failed to create destination file: "+err.Error())
			return nil, err
		}

		logger.Println(logger.DEBUG, true, "SSH: Copying "+file+" to "+dstFilePath)
		_, err = io.Copy(dstFile, bytes.NewReader(fileContents))
		if err != nil {
			log.Fatal("SSH: Failed to File Copy: ", err)
		}

		output, err := o.RunCmd("chmod +x " + dstFilePath)
		if err != nil {
			logger.Println(logger.ERROR, true, "SSH: Failed to run command: "+
				output+" (Error: "+err.Error())
			return nil, err
		}

		_ = dstFile.Close()
	}

	var BenchmarkList []model.Benchmark
	commands := "/tmp/milkyway.sh "
	types := []string{"cpus", "cpum", "memR", "memW", "fioR", "fioW", "dbR", "dbW"}

	for i, t := range types {
		logger.Printf(logger.DEBUG, true, "SSH: Benchmark Progressing - [%d/%d] %s...\n", i+1, len(types), t)
		output, err := o.RunCmd(commands + t)
		if err != nil {
			logger.Println(logger.ERROR, true, "SSH: Failed to run command: "+
				output+" (Error: "+err.Error())
			return nil, err
		}
		logger.Println(logger.DEBUG, true, "SSH: Benchmark Output: "+output)

		var response Response
		err = json.Unmarshal([]byte(output), &response)
		if err != nil {
			logger.Println(logger.ERROR, true, "SSH: Failed to unmarshal output: "+err.Error())
			return nil, err
		}

		benchmarkInfo := model.Benchmark{
			Type: t,
			Data: response.Data,
		}
		BenchmarkList = append(BenchmarkList, benchmarkInfo)
	}

	logger.Println(logger.DEBUG, true, BenchmarkList)

	return BenchmarkList, nil
}

func (o *SSH) getAuthMethods(connectionInfo model.ConnectionInfo) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	methods = o.tryPrivateKey(methods)

	methods = append(methods, ssh.PasswordCallback(func() (secret string, err error) {
		return connectionInfo.Password, nil
	}))

	return methods
}

func (o *SSH) tryPrivateKey(methods []ssh.AuthMethod) []ssh.AuthMethod {

	if !o.Options.IdentityFilePathProvided {
		if _, err := os.Stat(o.Options.IdentityFilePath); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("No ssh key at the default location %q found, skipping RSA authentication.\n", o.Options.IdentityFilePath)
			return methods
		}
	}

	callback := ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
		key, err := os.ReadFile(o.Options.IdentityFilePath)
		if err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(key)
		var passphraseMissingError *ssh.PassphraseMissingError
		isPassErr := errors.As(err, &passphraseMissingError)
		if isPassErr {
			signer, err = o.parsePrivateKeyWithPassphrase(key)
		}
		if err != nil {
			return nil, err
		}

		return []ssh.Signer{signer}, nil
	})

	return append(methods, callback)
}

func (o *SSH) parsePrivateKeyWithPassphrase(key []byte) (ssh.Signer, error) {
	return ssh.ParsePrivateKeyWithPassphrase(key, []byte{})
}

// RunCmd to SSH Server
func (o *SSH) RunCmd(cmd string) (string, error) {
	session, err := o.Options.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %s", err)
	}
	defer func() {
		_ = session.Close()
	}()

	var output bytes.Buffer
	var stderr bytes.Buffer

	session.Stdout = &output
	session.Stderr = &stderr

	if cmd != "" {
		logger.Println(logger.DEBUG, false, "SSH: ("+o.ConnectionInfo.IPAddress+") Running command: "+cmd)
		if err := session.Run(cmd); err != nil {
			return output.String() + "\n" + stderr.String(), err
		}
	}

	return output.String(), nil
}

func (o *SSH) Close() {
	if o.Options.session != nil {
		_ = o.Options.session.Close()
	}
	if o.Options.client != nil {
		_ = o.Options.client.Close()
	}
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

func StringToInterface(i string) interface{} {
	var x interface{}
	if err := json.Unmarshal([]byte(i), &x); err != nil {
		log.Printf("Error : %s\n", err)
	}
	return x
}
