package ssh

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jollaman999/utils/logger"

	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"

	"io"
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
	return Options{
		SSHPort:                  22,
		SSHUsername:              defaultUsername(),
		SSHPassword:              "",
		IdentityFilePath:         filepath.Join(homeDir, ".ssh", "id_rsa"),
		IdentityFilePathProvided: false,
	}
}

func (o *SSH) NewClientConn(connectionInfo model.ConnectionInfo) error {
	addr := fmt.Sprintf("%s:%d", connectionInfo.IPAddress, connectionInfo.SSHPort)

	sshConfig := &ssh.ClientConfig{
		User:            connectionInfo.User,
		Auth:            o.getAuthMethods(connectionInfo),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
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

func (o *SSH) RunBenchmark(connectionInfo model.ConnectionInfo, types string) (string, error) {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return "", err
	}
	defer o.Close()

	// SFTP Client 설정
	client, err := sftp.NewClient(o.Options.client)
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to SFTP Connect: "+err.Error())
		return "", err
	}
	defer client.Close()

	dstPath := "/tmp/"
	files := []string{"sourceFiles/milkyway", "sourceFiles/milkyway.sh"}

	if err := o.copyFilesToSFTP(client, files, dstPath); err != nil {
		return "", err
	}

	typesArr := o.getBenchmarkTypes(types)
	benchmarkData, err := o.BenchRunCmd(connectionInfo, typesArr)
	if err != nil {
		return "", err
	}

	benchmarkDataToJSON, _ := json.Marshal(benchmarkData)

	return string(benchmarkDataToJSON), nil
}

func (o *SSH) BenchRunCmd(connectionInfo model.ConnectionInfo, types []string) ([]model.Benchmark, error) {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return nil, err
	}
	defer o.Close()

	commands := "/tmp/milkyway.sh --run "
	var BenchmarkList []model.Benchmark

	for i, t := range types {
		if err := o.runBenchmarkCommand(commands+t, &BenchmarkList, t, i, len(types)); err != nil {
			return nil, err
		}
	}

	logger.Println(logger.DEBUG, true, "SSH: Benchmark Result : ", BenchmarkList)
	return BenchmarkList, nil
}

func (o *SSH) runBenchmarkCommand(cmd string, BenchmarkList *[]model.Benchmark, t string, i, total int) error {
	logger.Printf(logger.DEBUG, true, "SSH: Benchmark Progressing - [%d/%d] %s...\n", i+1, total, t)
	output, err := o.RunCmd(cmd)
	if err != nil {
		logger.Println(logger.DEBUG, true, "Failed to run command : ", err)
		return err
	}
	logger.Println(logger.DEBUG, true, "SSH: Benchmark Output: "+output)

	var response Response
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		logger.Println(logger.DEBUG, true, "Failed to unmarshal output", err)
		return err
	}

	*BenchmarkList = append(*BenchmarkList, model.Benchmark{Type: t, Data: response.Data})
	return nil
}

func (o *SSH) StopBenchmark(connectionInfo model.ConnectionInfo) error {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return err
	}
	defer o.Close()

	commands := "/tmp/milkyway.sh --stop"

	logger.Println(logger.DEBUG, true, "SSH: Benchmark Stopping..")
	if _, err := o.RunCmd(commands); err != nil {
		logger.Println(logger.DEBUG, true, "Failed to run command : ", err)
		return err
	}
	logger.Println(logger.DEBUG, true, "SSH: Benchmark Stopped")
	return nil
}

func (o *SSH) RunAgent(connectionInfo model.ConnectionInfo) (string, error) {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return "failed", err
	}
	defer o.Close()

	// SFTP Client 설정
	client, err := sftp.NewClient(o.Options.client)
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to SFTP Connect: "+err.Error())
		return "failed", err
	}
	defer client.Close()

	dstPath := "/tmp/"
	file := "sourceFiles/copyAgent.sh"

	if err := o.copyFileToSFTP(client, file, dstPath); err != nil {
		return "failed", err
	}

	commands := "/tmp/copyAgent.sh"
	logger.Printf(logger.DEBUG, true, "SSH: copyAgent Progressing...\n")
	if _, err := o.RunCmd("sudo " + commands); err != nil {
		logger.Println(logger.DEBUG, true, "Failed to run command : ", err)
		return "failed", err
	}

	return o.checkAgentStatus()
}

func (o *SSH) copyFilesToSFTP(client *sftp.Client, files []string, dstPath string) error {
	for _, file := range files {
		if err := o.copyFileToSFTP(client, file, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func (o *SSH) copyFileToSFTP(client *sftp.Client, file, dstPath string) error {
	fileContents, err := sourceFiles.ReadFile(file)
	if err != nil {
		logger.Println(logger.ERROR, true, "SSH: Failed to read source file: "+err.Error())
		return err
	}

	dstFilePath := filepath.Join(dstPath, filepath.Base(file))
	dstFile, err := client.Create(dstFilePath)
	if err != nil {
		logger.Println(logger.ERROR, true, "SSH: Failed to create destination file: "+err.Error())
		return err
	}
	defer dstFile.Close()

	logger.Println(logger.DEBUG, true, "SSH: Copying "+file+" to "+dstFilePath)
	if _, err := io.Copy(dstFile, bytes.NewReader(fileContents)); err != nil {
		logger.Println(logger.ERROR, true, "Failed to File Copy", err)
		return nil
	}

	if _, err := o.RunCmd("chmod +x " + dstFilePath); err != nil {
		logger.Println(logger.ERROR, true, "SSH: Failed to run command: ", err)
		return nil
	}

	return nil
}

func (o *SSH) getBenchmarkTypes(types string) []string {
	if types == "" {
		return []string{"cpus", "cpum", "memR", "memW", "fioR", "fioW"}
	}
	return strings.Split(types, ",")
}

func (o *SSH) checkAgentStatus() (string, error) {
	output, err := o.RunCmd("curl -o /dev/null -w '%{http_code}' -X GET http://localhost:8082/honeybee-agent/readyz -H 'accept: application/json'")
	if err != nil {
		logger.Println(logger.ERROR, true, "SSH: Failed to run command: "+
			output+" (Error: "+err.Error())
		return "failed", err
	}

	if output == "200" {
		return "success", nil
	}
	return "failed", nil
}

func (o *SSH) CheckKubernetes(connectionInfo model.ConnectionInfo) (bool, error) {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return false, err
	}
	defer o.Close()

	commands := "[ -f \"$HOME/.kube/config\" ] && echo \"true\" || echo \"false\""

	logger.Println(logger.DEBUG, true, "SSH: Kubernetes Checking...")
	chk, err := o.RunCmd(commands)
	if err != nil {
		logger.Println(logger.DEBUG, true, "Failed to run command : ", err)
		return false, err
	}

	chk = strings.TrimSpace(chk)
	return chk == "true", nil
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

	var output, stderr bytes.Buffer
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
