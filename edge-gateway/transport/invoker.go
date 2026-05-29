package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"edge-gateway/device"
)

// Invoker 将设备指令分发到具体的设备通信通道。
type Invoker interface {
	Invoke(ctx context.Context, dev device.Device, command device.Command) (device.Result, error)
}

type HTTPInvoker struct {
	Client *http.Client
}

func NewHTTPInvoker(client *http.Client) *HTTPInvoker {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPInvoker{Client: client}
}

func (i *HTTPInvoker) Invoke(ctx context.Context, dev device.Device, command device.Command) (device.Result, error) {
	dev = dev.Normalize()
	if dev.Protocol != device.ProtocolHTTPJSON {
		return device.Result{}, fmt.Errorf("device %q protocol %q is not supported", dev.ID, dev.Protocol)
	}
	if dev.Endpoint == "" {
		return device.Result{}, fmt.Errorf("device %q has no endpoint", dev.ID)
	}

	endpoint, err := url.Parse(dev.Endpoint)
	if err != nil {
		return device.Result{}, fmt.Errorf("parse endpoint for device %q: %w", dev.ID, err)
	}
	endpoint.Path = path.Join(endpoint.Path, "operations", command.Operation)

	payload := dev.NewInvokeRequest(command)
	body, err := json.Marshal(payload)
	if err != nil {
		return device.Result{}, fmt.Errorf("marshal command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return device.Result{}, fmt.Errorf("create invoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain")
	req.Header.Set("X-Device-ID", string(dev.ID))
	req.Header.Set("X-Operation", command.Operation)

	resp, err := i.Client.Do(req)
	if err != nil {
		return device.Result{}, fmt.Errorf("invoke device %q operation %q: %w", dev.ID, command.Operation, err)
	}
	defer resp.Body.Close()

	output, err := decodeDeviceResponse(resp)
	if err != nil {
		return device.Result{}, err
	}
	return device.Result{DeviceID: dev.ID, Operation: command.Operation, Output: output}, nil
}

func decodeDeviceResponse(resp *http.Response) (any, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read device response: %w", err)
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("device returned %s", resp.Status)
		}
		return nil, nil
	}

	var payload device.InvokeResponse
	jsonErr := json.Unmarshal(trimmed, &payload)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if jsonErr == nil && payload.Error != "" {
			return nil, fmt.Errorf("device returned %s: %s", resp.Status, payload.Error)
		}
		return nil, fmt.Errorf("device returned %s: %s", resp.Status, string(trimmed))
	}

	if jsonErr == nil {
		if payload.Error != "" {
			return nil, fmt.Errorf("device error: %s", payload.Error)
		}
		return payload.Output, nil
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/") || contentType == "" {
		return string(trimmed), nil
	}
	return nil, fmt.Errorf("decode device response: %w", jsonErr)
}
