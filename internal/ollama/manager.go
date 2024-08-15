package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type Manager struct {
	ServerStartedByUs bool
}

func NewManager() *Manager {
	return &Manager{ServerStartedByUs: false}
}

func (m *Manager) CheckInstallation() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}

func (m *Manager) InstallOllama() error {
	switch runtime.GOOS {
	case "darwin", "linux":
		fmt.Println("To install Ollama, run the following command in your terminal:")
		fmt.Println("curl https://ollama.ai/install.sh | sh")
	case "windows":
		fmt.Println("To install Ollama on Windows, please visit https://ollama.ai for installation instructions")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}

func (m *Manager) IsServerRunning() bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	_, err := client.Get("http://localhost:11434")
	return err == nil
}

func (m *Manager) StartServer() error {
	cmd := exec.Command("ollama", "serve")
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start Ollama server: %v", err)
	}

	// Wait for the server to start
	for i := 0; i < 10; i++ {
		if m.IsServerRunning() {
			m.ServerStartedByUs = true
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Ollama server did not start within the expected time")
}

func (m *Manager) StopServer() error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/F", "/IM", "ollama.exe")
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to stop Ollama server: %v", err)
		}
	} else {
		cmd := exec.Command("pkill", "ollama")
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to stop Ollama server: %v", err)
		}
	}
	m.ServerStartedByUs = false
	return nil
}

func (m *Manager) GetAvailableModels() ([]string, error) {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to get available models: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var models []string
	for _, model := range result.Models {
		models = append(models, model.Name)
	}

	return models, nil
}

func (m *Manager) SendMessage(model, message string) (string, error) {
	requestBody, err := json.Marshal(map[string]string{
		"model":  model,
		"prompt": message,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create request body: %v", err)
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Response, nil
}
