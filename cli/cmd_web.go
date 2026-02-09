/*
 * Copyright (C) 2026 Simone Pezzano
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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/util"
)

type executeRequest struct {
	Tools      *frags.ToolsConfig `json:"tools"`
	Plan       Plan               `json:"plan"`
	Parameters map[string]any     `json:"parameters"`
	Resources  map[string]string  `json:"resources"`
	Template   string             `json:"template"`
}

func (r *executeRequest) ToolsOrDefault(def frags.ToolsConfig) frags.ToolsConfig {
	if r.Tools == nil {
		return def
	}
	return *r.Tools
}

type Plan struct {
	*string
	*frags.SessionManager
}

// MarshalJSON implements custom JSON marshaling for Plan
func (p Plan) MarshalJSON() ([]byte, error) {
	// If string is set, marshal as string
	if p.string != nil {
		return json.Marshal(*p.string)
	}

	// If SessionManager is set, marshal as object
	if p.SessionManager != nil {
		return json.Marshal(p.SessionManager)
	}

	// If both are nil, marshal as null
	return json.Marshal(nil)
}

// UnmarshalJSON implements custom JSON unmarshaling for Plan
func (p *Plan) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		p.string = &str
		p.SessionManager = nil
		return nil
	}

	// If it's not a string, try to unmarshal as SessionManager
	var sm frags.SessionManager
	if err := json.Unmarshal(data, &sm); err != nil {
		return fmt.Errorf("failed to unmarshal Plan as string or SessionManager: %w", err)
	}

	p.SessionManager = &sm
	p.string = nil
	return nil
}

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "webserver related commands",
}

var errorHandler = func(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		_ = c.JSON(he.Code, echo.Map{"error": he.Message})
		return
	}
	_ = c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
}

var webExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "Run a Frags web server for the execute mode.",
	Long: `
Run a Frags web server for the execute mode. In the execute mode, you will be required to provide both the plan and the
tools configuration in the HTTP request. 
***WARNING***: this mode can easily turn into a security threat and allow RCE! Use this mode only in development or
safe environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		var logger *slog.Logger
		if debug {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		} else {
			logger = slog.Default()
		}
		e := echo.New()
		addRequestLoggerMiddleware(e, logger)
		if apiKey != "" {
			e.Use(apiKeyMiddleware)
		}
		e.HideBanner = true
		e.HTTPErrorHandler = errorHandler
		e.POST("/execute", func(c echo.Context) error {
			ctx := util.WithFragsContext(c.Request().Context(), 15*time.Minute)
			defer ctx.Cancel(nil)
			req := executeRequest{}
			if err := c.Bind(&req); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			sm := frags.NewSessionManager()
			if req.Plan.string != nil {
				if err := sm.FromYAML([]byte(*req.Plan.string)); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, err.Error())
				}
			}
			if req.Plan.SessionManager != nil {
				sm = *req.Plan.SessionManager
			}
			loader, err := filesMapToResourceLoader(req.Resources)
			if err != nil {
				return err
			}
			toolsConfig, err := readToolsFile()
			if err != nil {
				return err
			}
			if c.QueryParam("streaming") == "true" {
				level := log.ChannelLevel(c.QueryParam("level"))
				streamerLogger := log.NewStreamerLogger(logger, make(chan log.Event, 100), level)
				defer streamerLogger.Close()
				streamer := NewStreamer(c, streamerLogger)
				streamer.Start()
				result, err := execute(ctx, sm, req.Parameters, req.ToolsOrDefault(toolsConfig), loader, streamerLogger)
				time.Sleep(100 * time.Millisecond)
				if err != nil {
					return streamer.Finish(log.NewEvent(log.ErrorEventType, log.AppComponent).WithErr(err).WithLevel("err"))
				}
				output, _, err := dataOrRenderTemplate(c, req, result)
				if err != nil {
					return err
				}
				return streamer.Finish(log.NewEvent(log.ResultEventType, log.AppComponent).WithContent(output).WithLevel("info"))

			} else {
				streamerLogger := log.NewStreamerLogger(slog.Default(), nil, log.InfoChannelLevel)
				result, err := execute(ctx, sm, req.Parameters, req.ToolsOrDefault(toolsConfig), loader, streamerLogger)
				if err != nil {
					return err
				}
				output, isTemplate, err := dataOrRenderTemplate(c, req, result)
				if err != nil {
					return err
				}
				if isTemplate {
					c.Response().Header().Set("Content-Type", "text/markdown")
					return c.String(http.StatusOK, output.(string))
				} else {
					return c.JSON(http.StatusOK, result)
				}
			}
		})
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
			cmd.PrintErrln(err)
		}
	},
}

var webRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a Frags web server for the run mode.",
	Long: `
Run a Frags web server for the run mode. In the run mode, you will be required to provide the plan file name in the
request. The plans will be loaded from the selected directory. Tools settings will be global and governed by the
tools.json file.
***WARNING***: while way safer than "execute", also this mode offer possibility for exploitation. Check your plans
carefully and use this mode only in development or safe environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		var logger *slog.Logger
		if debug {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		} else {
			logger = slog.Default()
		}
		e := echo.New()
		addRequestLoggerMiddleware(e, logger)
		if apiKey != "" {
			e.Use(apiKeyMiddleware)
		}
		e.HideBanner = true
		e.HTTPErrorHandler = errorHandler
		initMCP(e)
		e.POST("/run/:file", func(c echo.Context) error {
			ctx := util.WithFragsContext(c.Request().Context(), 15*time.Minute)
			defer ctx.Cancel(nil)
			req := executeRequest{}
			if err := c.Bind(&req); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			fileRef, err := safePath(c.Param("file") + ".yaml")
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			planData, err := os.ReadFile(path.Join(rootDir, fileRef))
			if err != nil {
				return err
			}
			sm := frags.NewSessionManager()
			if err := sm.FromYAML(planData); err != nil {
				return err
			}
			toolsConfig, err := readToolsFile()
			if err != nil {
				return err
			}
			loader, err := filesMapToResourceLoader(req.Resources)
			if err != nil {
				return err
			}
			if c.QueryParam("streaming") == "true" {
				level := log.ChannelLevel(c.QueryParam("level"))
				streamerLogger := log.NewStreamerLogger(logger, make(chan log.Event, 100), level)
				defer streamerLogger.Close()
				streamer := NewStreamer(c, streamerLogger)
				streamer.Start()
				result, err := execute(ctx, sm, req.Parameters, req.ToolsOrDefault(toolsConfig), loader, streamerLogger)
				time.Sleep(100 * time.Millisecond)
				if err != nil {
					return streamer.Finish(log.NewEvent(log.ErrorEventType, log.AppComponent).WithErr(err).WithLevel("err"))
				}
				output, _, err := dataOrRenderLoadedTemplate(c, result)
				if err != nil {
					return err
				}
				return streamer.Finish(log.NewEvent(log.ResultEventType, log.AppComponent).WithContent(output).WithLevel("info"))

			} else {
				streamerLogger := log.NewStreamerLogger(logger, nil, log.InfoChannelLevel)
				result, err := execute(ctx, sm, req.Parameters, req.ToolsOrDefault(toolsConfig), loader, streamerLogger)
				if err != nil {
					return err
				}
				output, isTemplate, err := dataOrRenderLoadedTemplate(c, result)
				if err != nil {
					return err
				}
				if isTemplate {
					c.Response().Header().Set("Content-Type", "text/markdown")
					return c.String(http.StatusOK, output.(string))
				} else {
					return c.JSON(http.StatusOK, result)
				}

			}
		})
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
			cmd.PrintErrln(err)
		}
	},
}

func init() {
	webCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	webCmd.PersistentFlags().IntVarP(&port, "port", "", 8080, "port to listen on")
	webCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "", "", "a simple api key to protect the endpoint (it is expected in the x-api-key header)")

	webCmd.AddCommand(webExecuteCmd)

	webRunCmd.Flags().StringVarP(&rootDir, "root-directory", "", "", "directory where plans are stored")
	_ = webRunCmd.MarkFlagRequired("root-directory")

	webCmd.AddCommand(webRunCmd)
}

// addRequestLoggerMiddleware adds a middleware that logs each request.
func addRequestLoggerMiddleware(e *echo.Echo, log *slog.Logger) {
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				log.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				log.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))
}

func safePath(path string) (string, error) {
	path = filepath.Clean(path)
	err := checkTraversalPath(path)
	return path, err
}

// checkTraversalPath checks if the given filename is safe to use in a file system traversal.
func checkTraversalPath(filename string) error {
	if strings.Contains(filename, "..") ||
		strings.HasPrefix(filename, "/") ||
		strings.HasPrefix(filename, "\\") {
		return fmt.Errorf("invalid file name: %s", filename)
	}
	return nil
}

// filesMapToResourceLoader converts a map of files to a frags.ResourceLoader.
func filesMapToResourceLoader(files map[string]string) (resources.ResourceLoader, error) {
	loader := resources.NewBytesLoader()
	for k, v := range files {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
		loader.SetResource(resources.ResourceData{
			Identifier:  k,
			MediaType:   util.GetMediaType(k),
			ByteContent: decoded,
		})
	}
	return loader, nil
}

func apiKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := c.Request().Header.Get("x-api-key")
		if apiKey != key {
			return echo.NewHTTPError(http.StatusForbidden)
		}
		return next(c)
	}
}

func dataOrRenderTemplate(c echo.Context, req executeRequest, data *util.ProgMap) (any, bool, error) {
	if c.Request().Header.Get("Accept") == "text/markdown" && req.Template != "" {
		res, err := renderTemplate(req.Template, data)
		return string(res), true, err
	}
	return data, false, nil
}

func dataOrRenderLoadedTemplate(c echo.Context, data *util.ProgMap) (any, bool, error) {
	if c.Request().Header.Get("Accept") == "text/markdown" {
		fileRef, err := safePath(c.Param("file"))
		if err != nil {
			return nil, false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		template, err := os.ReadFile(path.Join(rootDir, fileRef+".md"))
		if err != nil {
			return nil, false, err
		}
		res, err := renderTemplate(string(template), data)
		return string(res), true, err
	}
	return data, false, nil
}
