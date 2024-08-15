package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "charmlama",
	Short: "CharmLlama - A charming CLI for Ollama models",
	Long:  `CharmLlama is a CLI-based chat application that allows users to interact with Ollama's open-source language models locally. It provides a user-friendly interface built with Go and Charm libraries, offering a rich terminal experience for AI-powered conversations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to CharmLlama!")
	},
}

func Execute() error {
	return rootCmd.Execute()
}
