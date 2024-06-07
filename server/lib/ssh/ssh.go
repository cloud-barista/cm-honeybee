package ssh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"

	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	Options Options
	// command string
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

func (o *SSH) NewClientConn(connectionInfo model.ConnectionInfo) ([]model.Benchmark, error) {
	addr := fmt.Sprintf("%s:%d", connectionInfo.IPAddress, connectionInfo.SSHPort)

	sshConfig := &ssh.ClientConfig{
		User:            connectionInfo.User,
		Auth:            o.getAuthMethods(connectionInfo),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}
	fmt.Println("SSH Connection Success. ")

	o.Options.client = client
	defer o.Close()

	// SFTP Client 설정
	sftp, err := sftp.NewClient(client)
	if err != nil {
		log.Fatal("Failed to SFTP Connect: ", err)
		return nil, err
	}
	defer sftp.Close()

	// 현재 작업 디렉토리 확인
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory: ", err)
	}
	log.Println("Current working directory: ", cwd)

	srcPath := filepath.Join(cwd, "lib", "ssh")
	dstPath := "/tmp/"
	fileName := "milkyway.sh"

	// log.Println(os.Getwd())

	// 소스 파일 절대 경로 얻기
	srcFilePath := filepath.Join(srcPath, fileName)

	// 소스 파일 열기
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		log.Fatal("Failed to open source file: ", err)
		return nil, err
	}

	// 파일을 다시 읽습니다.
	fileContents, err := io.ReadAll(srcFile)
	if err != nil {
		log.Fatal("Failed to read source file: ", err)
	}

	// 목적지 파일 절대 경로 얻기
	dstFilePath := filepath.Join(dstPath, fileName)

	// 목적지 파일 생성
	dstFile, err := sftp.Create(dstFilePath)
	if err != nil {
		log.Fatal("Failed to create destination file: ", err)
		return nil, err
	}
	log.Println("Current dstFile directory: ", dstFile)
	defer dstFile.Close()

	// 파일 복사
	_, err = io.Copy(dstFile, bytes.NewReader(fileContents))
	if err != nil {
		log.Fatal("Failed to File Copy: ", err)
	}

	var BenchmarkList []model.Benchmark
	commands := "/bin/bash /tmp/milkyway.sh "
	types := []string{"cpus", "cpum", "memR", "memW", "fioR", "fioW", "dbR", "dbW"}

	for i, t := range types {
		log.Printf("[%d/%d] %s - Benchmark Progressing...", i, len(types), t)
		output, err := o.RunCmd(commands + t)
		if err != nil {
			log.Fatal("Failed to run command: ", err)
		}

		var response Response
		err = json.Unmarshal([]byte(output), &response)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
		}

		benchmarkInfo := model.Benchmark{
			Type: t,
			Data: response.Data,
		}
		BenchmarkList = append(BenchmarkList, benchmarkInfo)
	}

	log.Println(BenchmarkList)

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
	defer session.Close()

	var output bytes.Buffer
	var stderr bytes.Buffer

	session.Stdout = &output
	session.Stderr = &stderr

	if cmd != "" {
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
