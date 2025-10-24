package ssh

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/server/lib/config"

	"github.com/jollaman999/utils/logger"

	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"

	"io"
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
	session *ssh.Session
	client  *ssh.Client
}

//go:embed sourceFiles/*
var sourceFiles embed.FS

func (o *SSH) NewClientConn(connectionInfo model.ConnectionInfo) error {
	addr := fmt.Sprintf("%s:%s", connectionInfo.IPAddress, connectionInfo.SSHPort)

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
		" Port: "+connectionInfo.SSHPort+", User: "+connectionInfo.User+")")

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
	defer func() {
		_ = client.Close()
	}()

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

	// Find the first complete JSON object
	var firstJSON string
	braceCount := 0
	inQuotes := false
	escapeNext := false

	for _, char := range output {
		if escapeNext {
			escapeNext = false
			firstJSON += string(char)
			continue
		}

		switch char {
		case '\\':
			escapeNext = true
			firstJSON += string(char)
		case '"':
			inQuotes = !inQuotes
			firstJSON += string(char)
		case '{':
			if !inQuotes {
				braceCount++
			}
			firstJSON += string(char)
		case '}':
			if !inQuotes {
				braceCount--
				if braceCount == 0 {
					firstJSON += string(char)
					// Found complete JSON object
					var response Response
					if err := json.Unmarshal([]byte(firstJSON), &response); err != nil {
						logger.Println(logger.DEBUG, true, "Failed to unmarshal output", err)
						return err
					}

					*BenchmarkList = append(*BenchmarkList, model.Benchmark{Type: t, Data: response.Data})
					return nil
				}
			}
			firstJSON += string(char)
		default:
			firstJSON += string(char)
		}
	}

	return fmt.Errorf("no valid JSON object found in output")
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

func (o *SSH) RunAgent(connectionInfo model.ConnectionInfo) error {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return err
	}
	defer o.Close()

	// SFTP Client 설정
	client, err := sftp.NewClient(o.Options.client)
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to SFTP Connect: "+err.Error())
		return err
	}
	defer func() {
		_ = client.Close()
	}()

	dstPath := "/tmp/"
	files := []string{"sourceFiles/busybox", "sourceFiles/copyAgent.sh"}

	if err = o.copyFilesToSFTP(client, files, dstPath); err != nil {
		return err
	}

	commands := "/tmp/copyAgent.sh"
	logger.Printf(logger.DEBUG, true, "SSH: copyAgent Progressing...\n")
	if _, err = o.RunCmd("sudo " + commands); err != nil {
		logger.Println(logger.ERROR, true, "Failed to run command : ", err)
		return err
	}

	commands = "rm -f /tmp/busybox && rm -f /tmp/copyAgent.sh"
	logger.Printf(logger.DEBUG, true, "SSH: Removing temporary files...\n")
	if _, err = o.RunCmd(commands); err != nil {
		logger.Println(logger.ERROR, true, "Failed to remove temporary files : ", err)
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
	defer func() {
		_ = dstFile.Close()
	}()

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

func (o *SSH) checkAgentStatus() error {
	tryCount := 30

	for i := 0; i < tryCount; i++ {
		output, err := o.RunCmd("curl -o /dev/null -w '%{http_code}' -X GET http://localhost:" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/honeybee-agent/readyz -H 'accept: application/json'")
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if output == "200" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return errors.New("agent health check failed")
}

func (o *SSH) SendGetRequestToAgent(connectionInfo model.ConnectionInfo, requestPath string) (string, error) {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return "", err
	}
	defer o.Close()

	output, err := o.RunCmd("curl -s -X GET http://localhost:" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/honeybee-agent" + requestPath + " -H 'accept: application/json'")
	if err != nil {
		return "", err
	}

	return output, nil
}

func (o *SSH) CheckKubernetes(connectionInfo model.ConnectionInfo) (bool, error) {
	if err := o.NewClientConn(connectionInfo); err != nil {
		return false, err
	}
	defer o.Close()

	commands := "[ -f \"/etc/kubernetes/admin.conf\" ] && echo \"true\" || echo \"false\""

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

	if connectionInfo.PrivateKey != "" && connectionInfo.PrivateKey != "-" {
		methods = o.tryPrivateKey(methods, connectionInfo)
	}

	if connectionInfo.Password != "" {
		methods = append(methods, ssh.PasswordCallback(func() (secret string, err error) {
			return connectionInfo.Password, nil
		}))
	}

	return methods
}

func (o *SSH) tryPrivateKey(methods []ssh.AuthMethod, connectionInfo model.ConnectionInfo) []ssh.AuthMethod {
	callback := ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
		privateKey := strings.ReplaceAll(connectionInfo.PrivateKey, "\\n", "\n")
		key := []byte(privateKey)

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			logger.Println(logger.ERROR, true, "Failed to parse private key: ", err)
			return nil, err
		}

		return []ssh.Signer{signer}, nil
	})

	return append(methods, callback)
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
