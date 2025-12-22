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
	engineGemini = "gemini"
	engineOllama = "ollama"
)

// supported output formats
const (
	formatTemplate = "template"
	formatYAML     = "yaml"
	formatJSON     = "json"
)

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
	UseKFormat               bool    `mapstructure:"USE_K_FORMAT"`
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
