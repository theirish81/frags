package main

import (
	"github.com/spf13/viper"
)

type Config struct {
	GeminiServiceAccountPath string `mapstructure:"GEMINI_SERVICE_ACCOUNT_PATH"`
	GeminiProjectID          string `mapstructure:"GEMINI_PROJECT_ID"`
	GeminiLocation           string `mapstructure:"GEMINI_LOCATION"`
	ParallelWorkers          int    `mapstructure:"PARALLEL_WORKERS"`
}

var cfg = Config{}

func main() {
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	_ = rootCmd.Execute()
}
