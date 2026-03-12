package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const AgentServerURL = "https://www.anwin.ai"

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	ProjectID string `json:"project_id"`
	WatchPath string `json:"watch_path"`
	Version   string `json:"version"`
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".anwin")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent.enc"), nil
}

func deriveMachineKey() []byte {
	hostname, _ := os.Hostname()
	var identity string
	switch runtime.GOOS {
	case "windows":
		identity = hostname + os.Getenv("USERNAME") + os.Getenv("COMPUTERNAME") + os.Getenv("USERPROFILE")
	case "darwin":
		identity = hostname + os.Getenv("USER") + os.Getenv("HOME") + "darwin"
	default:
		identity = hostname + os.Getenv("USER") + os.Getenv("HOME") + "linux"
	}
	hash := sha256.Sum256([]byte("anwin-agent-v1-salt::" + identity))
	return hash[:]
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("corrupted config")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, errors.New("decryption failed — run setup again")
	}
	return plain, nil
}

func Save(cfg *Config) error {
	cfg.ServerURL = AgentServerURL
	path, err := configFilePath()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	encrypted, err := encrypt(raw, deriveMachineKey())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(path, 0600)
	}
	return nil
}

func Load() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	encrypted, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("not configured")
		}
		return nil, err
	}
	raw, err := decrypt(encrypted, deriveMachineKey())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, errors.New("config is corrupt — run setup again")
	}
	cfg.ServerURL = AgentServerURL
	return &cfg, nil
}

func Delete() error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}