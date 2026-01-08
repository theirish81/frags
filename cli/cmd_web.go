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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
)

type executeRequest struct {
	Tools      frags.ToolsConfig `json:"tools"`
	Plan       string            `json:"plan"`
	Parameters map[string]any    `json:"parameters"`
	Resources  map[string]string `json:"resources"`
}

var web = &cobra.Command{
	Use:   "web",
	Short: "webserver related commands",
}

var errorHandler = func(err error, c echo.Context) {
	_ = c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
}

var webExecute = &cobra.Command{
	Use:   "execute",
	Short: "run a Frags web server for the execute mode.",
	Long: `
Run a Frags web server for the execute mode. In the execute mode, you will be required to provide both the plan and the
tools configuration in the HTTP request. 
***WARNING***: this mode can easily turn into a security threat and allow RCE! Use this mode only in development or
safe environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		var log *slog.Logger
		if debug {
			log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		} else {
			log = slog.Default()
		}
		e := echo.New()
		addRequestLoggerMiddleware(e, log)
		e.HideBanner = true
		e.HTTPErrorHandler = errorHandler
		e.POST("/execute", func(c echo.Context) error {
			req := executeRequest{}
			if err := c.Bind(&req); err != nil {
				return err
			}
			sm := frags.NewSessionManager()
			if err := sm.FromYAML([]byte(req.Plan)); err != nil {
				return err
			}
			loader, err := filesMapToResourceLoader(req.Resources)
			if err != nil {
				return err
			}
			result, err := execute(cmd.Context(), sm, req.Parameters, req.Tools, loader, log)
			if err != nil {
				return err
			}
			return c.JSON(http.StatusOK, result)
		})
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
			cmd.PrintErrln(err)
		}
	},
}

var webRun = &cobra.Command{
	Use:   "run",
	Short: "Run a Frags web server for the run mode.",
	Long: `
Run a Frags web server for the run mode. In the run mode, you will be required to provide the plan file name in the
request. The plans will be loaded from the selected directory. Tools settings will be global and governed by the
tools.json file.
***WARNING***: while way safer than "execute", also this mode offer possibility for exploitation. Check your plans
carefully and use this mode only in development or safe environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		var log *slog.Logger
		if debug {
			log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		} else {
			log = slog.Default()
		}
		e := echo.New()
		addRequestLoggerMiddleware(e, log)
		e.HideBanner = true
		e.HTTPErrorHandler = errorHandler
		e.POST("/run/:file", func(c echo.Context) error {
			req := executeRequest{}
			if err := c.Bind(&req); err != nil {
				return err
			}
			fileRef := filepath.Clean(c.Param("file") + ".yaml")
			if err := checkTraversalPath(fileRef); err != nil {
				return err
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
			result, err := execute(cmd.Context(), sm, req.Parameters, toolsConfig, loader, log)
			if err != nil {
				return err
			}
			return c.JSON(http.StatusOK, result)
		})
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
			cmd.PrintErrln(err)
		}
	},
}

func init() {
	web.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	web.PersistentFlags().IntVarP(&port, "port", "p", 8080, "port to listen on")

	web.AddCommand(webExecute)

	webRun.Flags().StringVarP(&rootDir, "root-directory", "", "", "directory where plans are stored")
	_ = webRun.MarkFlagRequired("root-directory")

	web.AddCommand(webRun)
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
func filesMapToResourceLoader(files map[string]string) (frags.ResourceLoader, error) {
	loader := frags.NewBytesLoader()
	for k, v := range files {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
		loader.SetResource(frags.ResourceData{
			Identifier: k,
			MediaType:  frags.GetMediaType(k),
			Data:       decoded,
		})
	}
	return loader, nil
}
