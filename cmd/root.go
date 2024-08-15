package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/marpit19/charmlama/internal/chat"
	"github.com/marpit19/charmlama/internal/models"
	"github.com/marpit19/charmlama/internal/ollama"
	"github.com/spf13/cobra"
)

var ollamaManager *ollama.Manager

var rootCmd = &cobra.Command{
	Use:   "charmlama",
	Short: "CharmLlama - A charming CLI for Ollama models",
	Long:  `CharmLlama is a CLI-based chat application that allows users to interact with Ollama's open-source language models locally. It provides a user-friendly interface built with Go and Charm libraries, offering a rich terminal experience for AI-powered conversations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ollamaManager = ollama.NewManager()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !ollamaManager.CheckInstallation() {
			fmt.Println("Ollama is not installed. Installing Ollama is required to use CharmLlama.")
			err := ollamaManager.InstallOllama()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			fmt.Println("Please install Ollama and run it before starting CharmLlama.")
			return
		}

		if !ollamaManager.IsServerRunning() {
			fmt.Println("Ollama server is not running. Would you like to start it? (yes/no)")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "yes" || response == "y" {
				fmt.Println("Starting Ollama server...")
				err := ollamaManager.StartServer()
				if err != nil {
					fmt.Printf("Failed to start Ollama server: %v\n", err)
					return
				}
				fmt.Println("Ollama server started successfully!")
			} else {
				fmt.Println("Ollama server is required to use CharmLlama.")
				fmt.Println("You can start it manually by running 'ollama serve' in a separate terminal.")
				return
			}
		}

		fmt.Println("Welcome to CharmLlama! Ollama server is running and ready.")

		for {
			// Get available models
			availableModels, err := ollamaManager.GetAvailableModels()
			if err != nil {
				fmt.Printf("Failed to get available models: %v\n", err)
				return
			}

			// Select model
			selectedModel, err := models.SelectModel(availableModels)
			if err != nil {
				if err == models.ErrUserQuit {
					fmt.Println("Exiting CharmLlama. Goodbye!")
					return
				}
				fmt.Printf("Failed to select model: %v\n", err)
				return
			}

			if selectedModel == "" {
				fmt.Println("No model selected. Exiting CharmLlama.")
				return
			}

			fmt.Printf("Selected model: %s\n", selectedModel)

			// Start chat interface
			chatInterface := chat.NewChatInterface(selectedModel, ollamaManager)
			returnToSelection, err := chatInterface.Run()
			if err != nil {
				fmt.Printf("Chat interface error: %v\n", err)
				return
			}

			if !returnToSelection {
				break
			}

			fmt.Println("Returning to model selection...")
		}

		// When exiting, stop the server if we started it
		defer func() {
			if ollamaManager.ServerStartedByUs {
				fmt.Println("Stopping Ollama server...")
				err := ollamaManager.StopServer()
				if err != nil {
					fmt.Printf("Failed to stop Ollama server: %v\n", err)
				} else {
					fmt.Println("Ollama server stopped successfully.")
				}
			}
		}()
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Ollama server",
	Run: func(cmd *cobra.Command, args []string) {
		if !ollamaManager.IsServerRunning() {
			fmt.Println("Ollama server is not running.")
			return
		}

		fmt.Println("Stopping Ollama server...")
		err := ollamaManager.StopServer()
		if err != nil {
			fmt.Printf("Failed to stop Ollama server: %v\n", err)
		} else {
			fmt.Println("Ollama server stopped successfully.")
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
