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

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"cloud.google.com/go/auth/credentials"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/gemini"
	"github.com/theirish81/frags/ollama"
	"google.golang.org/genai"
)

// initAi initializes the AI engine based on the configuration.
func initAi(log *slog.Logger) (frags.Ai, error) {
	switch cfg.guessAi() {
	case engineGemini:
		client, err := newGeminiClient()
		if err != nil {
			return nil, err
		}
		return gemini.NewAI(client, gemini.Config{
			Temperature: cfg.Temperature,
			TopK:        cfg.TopK,
			TopP:        cfg.TopP,
			Model:       cfg.Model,
		}, log), nil
	case engineOllama:
		return ollama.NewAI(cfg.OllamaBaseURL, ollama.Config{
			Temperature: cfg.Temperature,
			TopK:        cfg.TopK,
			TopP:        cfg.TopP,
			Model:       cfg.Model,
			NumPredict:  cfg.NumPredict,
		}, log), nil
	default:
		return nil, errors.New("No AI is fully configured. Check your .env file")
	}
}

// newGeminiClient constructs a genai client using the configured service account.
func newGeminiClient() (*genai.Client, error) {
	credsBytes, err := os.ReadFile(cfg.GeminiServiceAccountPath)
	if err != nil {
		return nil, err
	}
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		CredentialsJSON: credsBytes,
	})
	if err != nil {
		return nil, err
	}
	return genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:     cfg.GeminiProjectID,
		Location:    cfg.GeminiLocation,
		Credentials: creds,
		Backend:     genai.BackendVertexAI,
	})
}
