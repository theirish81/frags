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

package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type HttpClient struct {
	http.Client
	baseURL string
}

type Transport struct {
	defaultRoundtripper http.RoundTripper
	apiKey              string
}

func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.defaultRoundtripper.RoundTrip(req)
}

func NewTransport(apiKey string) *Transport {
	return &Transport{
		apiKey:              apiKey,
		defaultRoundtripper: http.DefaultTransport,
	}
}

func NewHttpClient(baseURL string, apiKey string) *HttpClient {
	return &HttpClient{
		baseURL: baseURL,
		Client: http.Client{
			Timeout:   5 * time.Minute,
			Transport: NewTransport(apiKey),
		},
	}
}

func (c *HttpClient) PostResponses(ctx context.Context, content any) (Response, error) {
	response := Response{}
	data, err := json.Marshal(content)
	if err != nil {
		return response, err
	}
	reader := bytes.NewReader(data)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/responses", reader)
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	if err != nil {
		return response, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	data, err = io.ReadAll(res.Body)
	if err != nil {
		return response, err
	}
	if res.StatusCode >= 400 {
		return response, errors.New(string(data))
	}

	err = json.Unmarshal(data, &response)
	return response, err
}

func (c *HttpClient) FileUpload(ctx context.Context, filename string, content []byte) (FileDescriptor, error) {
	fd := FileDescriptor{}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("purpose", "assistants"); err != nil {
		return fd, err
	}

	if err := writer.WriteField("expires_after[anchor]", "created_at"); err != nil {
		return fd, err
	}

	if err := writer.WriteField("expires_after[seconds]", "3600"); err != nil {
		return fd, err
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fd, err
	}

	if _, err := part.Write(content); err != nil {
		return fd, err
	}

	if err := writer.Close(); err != nil {
		return fd, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/files", &body)
	if err != nil {
		return fd, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.Client.Do(req)
	if err != nil {
		return fd, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	data, err := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return fd, errors.New(string(data))
	}

	if err != nil {
		return fd, err
	}
	err = json.Unmarshal(data, &fd)
	return fd, err
}
