package main

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/theirish81/frags/gemini"
)

const engineGemini = "gemini"
const engineOllama = "ollama"

type Config struct {
	GeminiServiceAccountPath string  `mapstructure:"GEMINI_SERVICE_ACCOUNT_PATH"`
	GeminiProjectID          string  `mapstructure:"GEMINI_PROJECT_ID"`
	GeminiLocation           string  `mapstructure:"GEMINI_LOCATION"`
	ParallelWorkers          int     `mapstructure:"PARALLEL_WORKERS"`
	OllamaBaseURL            string  `mapstructure:"OLLAMA_BASE_URL"`
	Model                    string  `mapstructure:"MODEL"`
	AiEngine                 string  `mapstructure:"AI_ENGINE"`
	Temperature              float32 `mapstructure:"TEMPERATURE"`
	TopK                     float32 `mapstructure:"TOP_K"`
	TopP                     float32 `mapstructure:"TOP_P"`
	NumPredict               int     `mapstructure:"NUM_PREDICT"`
}

func (c Config) guessAi() string {
	switch strings.ToLower(c.AiEngine) {
	case engineOllama:
		return engineOllama
	case engineGemini:
		return engineGemini
	}
	if c.OllamaBaseURL != "" && c.Model != "" {
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
		data := make(map[string]any)
		defCfg := gemini.DefaultConfig()
		cfg.AiEngine = "gemini"
		cfg.Model = defCfg.Model
		cfg.TopK = defCfg.TopK
		cfg.TopP = defCfg.TopP
		cfg.Temperature = defCfg.Temperature
		cfg.ParallelWorkers = 1
		cfg.NumPredict = 1024
		_ = mapstructure.Decode(&cfg, &data)
		_ = viper.MergeConfigMap(data)
		viper.SetConfigType("env")
		_ = viper.WriteConfigAs(".env")
		fmt.Println("an empty .env file was created, please fill it out and try again.")
		_ = rootCmd.Help()
		return
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	_ = rootCmd.Execute()
}
