package main

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const engineGemini = "gemini"
const engineOllama = "ollama"

type Config struct {
	GeminiServiceAccountPath string `mapstructure:"GEMINI_SERVICE_ACCOUNT_PATH"`
	GeminiProjectID          string `mapstructure:"GEMINI_PROJECT_ID"`
	GeminiLocation           string `mapstructure:"GEMINI_LOCATION"`
	ParallelWorkers          int    `mapstructure:"PARALLEL_WORKERS"`
	OllamaBaseURL            string `mapstructure:"OLLAMA_BASE_URL"`
	OllamaModel              string `mapstructure:"OLLAMA_MODEL"`
	AiEngine                 string `mapstructure:"AI_ENGINE"`
}

func (c Config) guessAi() string {
	switch strings.ToLower(c.AiEngine) {
	case engineOllama:
		return engineOllama
	case engineGemini:
		return engineGemini
	}
	if c.OllamaBaseURL != "" && c.OllamaModel != "" {
		return engineOllama
	}
	if c.GeminiServiceAccountPath != "" && c.GeminiProjectID != "" && c.GeminiLocation != "" {
		return engineGemini
	}
	return ""
}

var cfg = Config{}

func main() {
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("error reading .env file: ", err)
		return
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	_ = rootCmd.Execute()
}
