package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type AgentConfig struct {
	ServerURL string `json:"serverUrl"`
	Token     string `json:"token"`
	WatchPath string `json:"watchPath"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".anwin")
}

func configPath() string {
	return filepath.Join(configDir(), "agent.enc")
}

func Save(cfg *AgentConfig) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot serialize config: %w", err)
	}

	encrypted, err := encrypt(data, deriveKey())
	if err != nil {
		return fmt.Errorf("cannot encrypt config: %w", err)
	}

	if err := os.WriteFile(configPath(), []byte(hex.EncodeToString(encrypted)), 0600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}

	return nil
}

func Load() (*AgentConfig, error) {
	raw, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("config not found: %w", err)
	}

	ciphertext, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("corrupted config: %w", err)
	}

	plaintext, err := decrypt(ciphertext, deriveKey())
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt config: %w", err)
	}

	var cfg AgentConfig
	if err := json.Unmarshal(plaintext, &cfg); err != nil {
		return nil, fmt.Errorf("corrupted config data: %w", err)
	}

	return &cfg, nil
}

func Delete() error {
	return os.Remove(configPath())
}

func MachineFingerprint() string {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	hash := sha256.Sum256([]byte(hostname + "|" + username))
	return hex.EncodeToString(hash[:])
}

func DetectPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "MAC"
	case "windows":
		return "WINDOWS"
	default:
		return "LINUX"
	}
}

func deriveKey() []byte {
	fingerprint := MachineFingerprint()
	hash := sha256.Sum256([]byte("anwin-agent-key-v2|" + fingerprint))
	return hash[:]
}

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}
