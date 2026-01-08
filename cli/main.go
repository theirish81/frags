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
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/theirish81/frags/gemini"
)

func main() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		data := make(map[string]any)
		defCfg := gemini.DefaultConfig()
		cfg.AiEngine = "gemini"
		cfg.GeminiLocation = "global"
		cfg.Model = defCfg.Model
		cfg.TopK = defCfg.TopK
		cfg.TopP = defCfg.TopP
		cfg.Temperature = defCfg.Temperature
		cfg.OllamaBaseURL = "http://localhost:11434"
		cfg.ParallelWorkers = 1
		cfg.NumPredict = 1024
		cfg.UseKFormat = false
		cfg.ChatGptBaseURL = "https://api.openai.com/v1"
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
