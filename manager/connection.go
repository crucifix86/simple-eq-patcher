package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ConnectionProfile stores SSH connection details
type ConnectionProfile struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	RemotePath string `json:"remote_path"`
}

// ConnectionManager handles SSH/SFTP connections
type ConnectionManager struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	profile    *ConnectionProfile
	connected  bool
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connected: false,
	}
}

// Connect establishes SSH and SFTP connections
func (cm *ConnectionManager) Connect(profile *ConnectionProfile) error {
	// Create SSH client config
	config := &ssh.ClientConfig{
		User: profile.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(profile.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add proper host key verification
		Timeout:         15 * time.Second,
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%s", profile.Host, profile.Port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}

	cm.sshClient = sshClient
	cm.sftpClient = sftpClient
	cm.profile = profile
	cm.connected = true

	return nil
}

// Disconnect closes SSH and SFTP connections
func (cm *ConnectionManager) Disconnect() error {
	if cm.sftpClient != nil {
		cm.sftpClient.Close()
	}
	if cm.sshClient != nil {
		cm.sshClient.Close()
	}
	cm.connected = false
	return nil
}

// IsConnected returns connection status
func (cm *ConnectionManager) IsConnected() bool {
	return cm.connected
}

// ListRemoteDir lists files in a remote directory
func (cm *ConnectionManager) ListRemoteDir(path string) ([]string, error) {
	if !cm.connected {
		return nil, fmt.Errorf("not connected")
	}

	files, err := cm.sftpClient.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, file := range files {
		filenames = append(filenames, file.Name())
	}

	return filenames, nil
}

// ExecuteCommand runs a command on the remote server
func (cm *ConnectionManager) ExecuteCommand(command string) (string, error) {
	if !cm.connected {
		return "", fmt.Errorf("not connected")
	}

	session, err := cm.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v", err)
	}

	return string(output), nil
}

// UploadFile uploads a file to the remote server
func (cm *ConnectionManager) UploadFile(localPath, remotePath string) error {
	if !cm.connected {
		return fmt.Errorf("not connected")
	}

	// Read local file
	data, err := ioutil.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %v", err)
	}

	// Create remote file
	remoteFile, err := cm.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %v", err)
	}
	defer remoteFile.Close()

	// Write data
	_, err = remoteFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to remote file: %v", err)
	}

	return nil
}

// UploadFileResumable uploads a file with resume capability
func (cm *ConnectionManager) UploadFileResumable(localPath, remotePath string, progressCallback func(int64, int64)) error {
	if !cm.connected {
		return fmt.Errorf("not connected")
	}

	// Read local file
	data, err := ioutil.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %v", err)
	}

	totalSize := int64(len(data))

	// Check if remote file exists (for resume)
	remoteInfo, err := cm.sftpClient.Stat(remotePath)
	var startOffset int64 = 0
	if err == nil {
		// File exists, resume from this point
		startOffset = remoteInfo.Size()
	}

	// Open remote file (append if resuming, create if new)
	var remoteFile *sftp.File
	if startOffset > 0 {
		remoteFile, err = cm.sftpClient.OpenFile(remotePath, os.O_WRONLY|os.O_APPEND)
	} else {
		remoteFile, err = cm.sftpClient.Create(remotePath)
	}
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer remoteFile.Close()

	// Write data in chunks with progress reporting
	chunkSize := 65536 // 64KB chunks
	for i := startOffset; i < totalSize; i += int64(chunkSize) {
		end := i + int64(chunkSize)
		if end > totalSize {
			end = totalSize
		}

		_, err := remoteFile.Write(data[i:end])
		if err != nil {
			return fmt.Errorf("failed to write chunk: %v", err)
		}

		if progressCallback != nil {
			progressCallback(end, totalSize)
		}
	}

	return nil
}

// CreateRemoteDir creates a directory on the remote server
func (cm *ConnectionManager) CreateRemoteDir(path string) error {
	if !cm.connected {
		return fmt.Errorf("not connected")
	}

	return cm.sftpClient.MkdirAll(path)
}

// DownloadFile downloads a file from the remote server
func (cm *ConnectionManager) DownloadFile(remotePath, localPath string) error {
	if !cm.connected {
		return fmt.Errorf("not connected")
	}

	// Open remote file
	remoteFile, err := cm.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer remoteFile.Close()

	// Read remote file
	data, err := ioutil.ReadAll(remoteFile)
	if err != nil {
		return fmt.Errorf("failed to read remote file: %v", err)
	}

	// Write to local file
	err = ioutil.WriteFile(localPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write local file: %v", err)
	}

	return nil
}
