/*
 * Copyright (C) 2025 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import "strings"

// supported AI engines
const (
	engineGemini  = "gemini"
	engineOllama  = "ollama"
	engineChatgpt = "chatgpt"
)

// supported output formats
const (
	formatTemplate = "template"
	formatYAML     = "yaml"
	formatJSON     = "json"
)

type Config struct {
	GeminiServiceAccountPath string  `mapstructure:"GEMINI_SERVICE_ACCOUNT_PATH" yaml:"GEMINI_SERVICE_ACCOUNT_PATH"`
	GeminiProjectID          string  `mapstructure:"GEMINI_PROJECT_ID" yaml:"GEMINI_PROJECT_ID"`
	GeminiLocation           string  `mapstructure:"GEMINI_LOCATION" yaml:"GEMINI_LOCATION"`
	ParallelWorkers          int     `mapstructure:"PARALLEL_WORKERS" yaml:"PARALLEL_WORKERS"`
	OllamaBaseURL            string  `mapstructure:"OLLAMA_BASE_URL" yaml:"OLLAMA_BASE_URL"`
	Model                    string  `mapstructure:"MODEL" yaml:"MODEL"`
	AiEngine                 string  `mapstructure:"AI_ENGINE" yaml:"AI_ENGINE"`
	Temperature              float32 `mapstructure:"TEMPERATURE" yaml:"TEMPERATURE"`
	TopK                     float32 `mapstructure:"TOP_K" yaml:"TOP_K"`
	TopP                     float32 `mapstructure:"TOP_P" yaml:"TOP_P"`
	NumPredict               int     `mapstructure:"NUM_PREDICT" yaml:"NUM_PREDICT"`
	UseKFormat               bool    `mapstructure:"USE_K_FORMAT" yaml:"USE_K_FORMAT"`
	ChatGptApiKey            string  `mapstructure:"CHATGPT_API_KEY" yaml:"CHATGPT_API_KEY"`
	ChatGptBaseURL           string  `mapstructure:"CHATGPT_BASE_URL" yaml:"CHATGPT_BASE_URL"`
}

// guessAi tries to guess the AI engine based on the configuration.
func (c Config) guessAi() string {
	switch strings.ToLower(c.AiEngine) {
	case engineOllama:
		return engineOllama
	case engineGemini:
		return engineGemini
	case engineChatgpt:
		return engineChatgpt
	}
	if c.OllamaBaseURL != "" && c.Model != "" {
		return engineOllama
	}
	if c.GeminiServiceAccountPath != "" && c.GeminiProjectID != "" && c.GeminiLocation != "" {
		return engineGemini
	}
	if c.ChatGptApiKey != "" && c.ChatGptBaseURL != "" {
		return engineChatgpt
	}
	return ""
}

var cfg = Config{}
