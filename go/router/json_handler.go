package router

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type schemaResolver interface {
	Resolve(ctx context.Context, refStr string) (protoreflect.MessageType, error)
}

type JSONHandler struct {
	registry *registry.UnifiedRegistry
	resolver schemaResolver
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type searchRegistryArgs struct {
	Query string `json:"query"`
}

type resolveSchemaArgs struct {
	BsrRef string `json:"bsr_ref"`
}

type callToolArgs struct {
	BsrRef    string          `json:"bsr_ref"`
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
}

func NewJSONHandler(reg *registry.UnifiedRegistry, resolver schemaResolver) *JSONHandler {
	return &JSONHandler{
		registry: reg,
		resolver: resolver,
	}
}

func (h *JSONHandler) Handle(rw io.ReadWriter) error {
	enc := json.NewEncoder(rw)
	reader := bufio.NewReader(rw)

	for {
		var req jsonRPCRequest
		if err := readJSONRPCRequest(reader, &req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to decode JSON-RPC request: %w", err)
		}

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "proto-mcp-dual-server",
					"version": "0.1.0",
				},
			}
		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": h.listTools(),
			}
		case "tools/call":
			result, err := h.handleToolCall(context.Background(), req.Params)
			if err != nil {
				resp.Error = jsonRPCError{
					Code:    -32603,
					Message: err.Error(),
				}
			} else {
				resp.Result = result
			}
		default:
			resp.Error = jsonRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("unsupported method: %s", req.Method),
			}
		}

		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("failed to encode JSON-RPC response: %w", err)
		}
	}
}

func readJSONRPCRequest(reader *bufio.Reader, req *jsonRPCRequest) error {
	if hasContentLengthHeaderReader(reader) {
		body, err := readContentLengthBody(reader)
		if err != nil {
			return err
		}
		return json.Unmarshal(body, req)
	}

	dec := json.NewDecoder(reader)
	return dec.Decode(req)
}

func hasContentLengthHeaderReader(reader *bufio.Reader) bool {
	for {
		peek, err := reader.Peek(1)
		if err != nil {
			return false
		}
		if !isWhitespace(peek[0]) {
			break
		}
		if _, err := reader.ReadByte(); err != nil {
			return false
		}
	}

	peek, err := reader.Peek(len("Content-Length:"))
	if err != nil {
		return false
	}
	return strings.EqualFold(string(peek), "Content-Length:")
}

func readContentLengthBody(reader *bufio.Reader) ([]byte, error) {
	const maxContentLength = 100 * 1024 * 1024 // 100MB limit to prevent DoS

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}

	lengthStr := headers["content-length"]
	if lengthStr == "" {
		return nil, fmt.Errorf("content-length header missing")
	}
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid content-length: %w", err)
	}
	if length <= 0 {
		return nil, fmt.Errorf("invalid content-length: %d", length)
	}
	if length > maxContentLength {
		return nil, fmt.Errorf("content-length %d exceeds maximum of %d bytes", length, maxContentLength)
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (h *JSONHandler) listTools() []map[string]interface{} {
	if h.registry == nil {
		return []map[string]interface{}{}
	}

	// Get all tools from the registry
	protoTools := h.registry.List("")
	result := make([]map[string]interface{}, 0, len(protoTools))

	for _, tool := range protoTools {
		// Convert proto Tool to JSON format
		toolJSON := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		}

		// Add BSR ref as metadata if present
		if bsrRef := toolBsrRef(tool); bsrRef != "" {
			toolJSON["bsr_ref"] = bsrRef
		}

		// For JSON clients, we provide a minimal schema that indicates
		// they should use resolve_schema to get the full schema on-demand
		toolJSON["inputSchema"] = map[string]interface{}{
			"type": "object",
			"description": "Use resolve_schema tool to fetch the full schema for this tool's BSR reference",
		}

		result = append(result, toolJSON)
	}

	return result
}

func toolBsrRef(tool *mcp.Tool) string {
	if tool == nil {
		return ""
	}
	if ref, ok := tool.SchemaSource.(*mcp.Tool_BsrRef); ok {
		return ref.BsrRef
	}
	return ""
}

func (h *JSONHandler) handleToolCall(ctx context.Context, rawParams json.RawMessage) (map[string]interface{}, error) {
	var params toolCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}

	switch params.Name {
	case "search_registry":
		return h.handleSearchRegistry(ctx, params.Arguments)
	case "resolve_schema":
		return h.handleResolveSchema(ctx, params.Arguments)
	case "call_tool":
		return h.handleCallTool(ctx, params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool name: %s", params.Name)
	}
}

func (h *JSONHandler) handleSearchRegistry(ctx context.Context, rawArgs json.RawMessage) (map[string]interface{}, error) {
	if h.registry == nil {
		return nil, fmt.Errorf("registry is not configured")
	}

	var args searchRegistryArgs
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid search_registry args: %w", err)
		}
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		query = "analytics"
	}

	candidates, err := h.registry.SearchRegistry(ctx, query)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query":      query,
		"candidates": candidates,
	})
	if err != nil {
		return nil, err
	}

	return textResult(string(payload)), nil
}

func (h *JSONHandler) handleResolveSchema(ctx context.Context, rawArgs json.RawMessage) (map[string]interface{}, error) {
	if h.resolver == nil {
		return nil, fmt.Errorf("schema resolver is not configured")
	}

	var args resolveSchemaArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid resolve_schema args: %w", err)
	}
	if strings.TrimSpace(args.BsrRef) == "" {
		return nil, fmt.Errorf("bsr_ref is required")
	}

	msgType, err := h.resolver.Resolve(ctx, args.BsrRef)
	if err != nil {
		return nil, err
	}

	schema := messageSchema(msgType.Descriptor(), map[protoreflect.FullName]bool{})
	payload, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	return textResult(string(payload)), nil
}

func (h *JSONHandler) handleCallTool(ctx context.Context, rawArgs json.RawMessage) (map[string]interface{}, error) {
	if h.registry == nil {
		return nil, fmt.Errorf("registry is not configured")
	}
	if h.resolver == nil {
		return nil, fmt.Errorf("schema resolver is not configured")
	}

	var args callToolArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid call_tool args: %w", err)
	}
	args.BsrRef = strings.TrimSpace(args.BsrRef)
	if args.BsrRef == "" {
		return nil, fmt.Errorf("bsr_ref is required")
	}

	msgType, err := h.resolver.Resolve(ctx, args.BsrRef)
	if err != nil {
		return nil, err
	}

	msg := msgType.New().Interface()
	if len(args.Arguments) == 0 {
		args.Arguments = []byte("{}")
	}
	if err := protojson.Unmarshal(args.Arguments, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var result *mcp.ToolResult
	if strings.TrimSpace(args.ToolName) != "" {
		result, err = h.registry.Call(ctx, args.ToolName, payload)
	} else {
		result, err = h.registry.CallByBsrRef(ctx, args.BsrRef, payload)
	}
	if err != nil {
		return nil, err
	}

	return toolResult(result), nil
}

func textResult(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text,
			},
		},
	}
}

func toolResult(result *mcp.ToolResult) map[string]interface{} {
	if result == nil {
		return textResult("")
	}

	content := make([]map[string]interface{}, 0, len(result.Content))
	for _, item := range result.Content {
		switch c := item.Content.(type) {
		case *mcp.ToolContent_Text:
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": c.Text,
			})
		case *mcp.ToolContent_Image:
			content = append(content, map[string]interface{}{
				"type": "image",
				"data": c.Image,
			})
		}
	}

	return map[string]interface{}{
		"content": content,
	}
}

func messageSchema(desc protoreflect.MessageDescriptor, seen map[protoreflect.FullName]bool) map[string]interface{} {
	if desc == nil {
		return map[string]interface{}{"type": "object"}
	}
	if seen[desc.FullName()] {
		return map[string]interface{}{"type": "object"}
	}
	seen[desc.FullName()] = true

	properties := make(map[string]interface{})
	fields := desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		properties[field.JSONName()] = fieldSchema(field, seen)
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
}

func fieldSchema(field protoreflect.FieldDescriptor, seen map[protoreflect.FullName]bool) map[string]interface{} {
	if field.IsList() {
		return map[string]interface{}{
			"type":  "array",
			"items": scalarSchema(field, seen),
		}
	}
	if field.IsMap() {
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": scalarSchema(field.MapValue(), seen),
		}
	}
	return scalarSchema(field, seen)
}

func scalarSchema(field protoreflect.FieldDescriptor, seen map[protoreflect.FullName]bool) map[string]interface{} {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return map[string]interface{}{"type": "boolean"}
	case protoreflect.StringKind:
		return map[string]interface{}{"type": "string"}
	case protoreflect.BytesKind:
		return map[string]interface{}{
			"type":             "string",
			"contentEncoding":  "base64",
			"contentMediaType": "application/octet-stream",
		}
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return map[string]interface{}{"type": "integer"}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return map[string]interface{}{"type": "number"}
	case protoreflect.EnumKind:
		enum := field.Enum()
		values := make([]string, 0, enum.Values().Len())
		for i := 0; i < enum.Values().Len(); i++ {
			values = append(values, string(enum.Values().Get(i).Name()))
		}
		return map[string]interface{}{
			"type": "string",
			"enum": values,
		}
	case protoreflect.MessageKind:
		fullName := field.Message().FullName()
		switch fullName {
		case "google.protobuf.Timestamp":
			return map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			}
		case "google.protobuf.Duration":
			return map[string]interface{}{"type": "string"}
		case "google.protobuf.Struct":
			return map[string]interface{}{"type": "object"}
		case "google.protobuf.Any":
			return map[string]interface{}{"type": "object"}
		case "google.protobuf.ListValue":
			return map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{},
			}
		case "google.protobuf.Value":
			return map[string]interface{}{"type": "object"}
		default:
			return messageSchema(field.Message(), seen)
		}
	default:
		return map[string]interface{}{"type": "string"}
	}
}
