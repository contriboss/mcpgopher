package transport

import (
	"context"
	"encoding/json"
)

// Interface for the transport layer.
type Interface interface {
	// Start the connection. Start should only be called once.
	Start(ctx context.Context) error

	// SendRequest sends a json RPC request and returns the response synchronously.
	SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error)

	// SendNotification sends a json RPC Notification to the server.
	SendNotification(ctx context.Context, notification JSONRPCNotification) error

	// SetNotificationHandler sets the handler for notifications.
	// Any notification before the handler is set will be discarded.
	SetNotificationHandler(handler func(notification JSONRPCNotification))

	// Close the connection.
	Close() error
}

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"error"`
}

type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		AdditionalFields map[string]interface{} `json:"-"`
	} `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (n *JSONRPCNotification) UnmarshalJSON(data []byte) error {
	type alias JSONRPCNotification
	aux := struct {
		Params json.RawMessage `json:"params,omitempty"`
		*alias
	}{
		alias: (*alias)(n),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	if len(aux.Params) > 0 {
		var additionalFields map[string]interface{}
		if err := json.Unmarshal(aux.Params, &additionalFields); err != nil {
			return err
		}
		n.Params.AdditionalFields = additionalFields
	}
	
	return nil
}