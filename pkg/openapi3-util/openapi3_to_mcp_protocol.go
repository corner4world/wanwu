package openapi3_util

import (
	"context"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/getkin/kin-openapi/openapi3"
)

func Schema2MCPProtocolTools(ctx context.Context, schema []byte) ([]*protocol.Tool, error) {
	doc, err := LoadFromData(ctx, schema)
	if err != nil {
		return nil, err
	}
	return Doc2MCPProtocolTools(doc)
}

func Schema2MCPProtocolTool(ctx context.Context, schema []byte, operationID string) (*protocol.Tool, error) {
	doc, err := LoadFromData(ctx, schema)
	if err != nil {
		return nil, err
	}
	return Doc2MCPProtocolTool(doc, operationID)
}

func Doc2MCPProtocolTools(doc *openapi3.T) ([]*protocol.Tool, error) {
	var rets []*protocol.Tool
	for _, pathItem := range doc.Paths {
		for _, operation := range pathItem.Operations() {
			rets = append(rets, Operation2MCPProtocolTool(operation))
		}
	}
	return rets, nil
}

func Doc2MCPProtocolTool(doc *openapi3.T, operationID string) (*protocol.Tool, error) {
	var exist bool
	var ret *protocol.Tool
	for _, pathItem := range doc.Paths {
		for _, operation := range pathItem.Operations() {
			if operation.OperationID != operationID {
				continue
			}
			exist = true
			ret = Operation2MCPProtocolTool(operation)
			break
		}
	}
	if !exist {
		return nil, fmt.Errorf("opentionID(%v) not found", operationID)
	}
	return ret, nil
}

func Operation2MCPProtocolTool(operation *openapi3.Operation) *protocol.Tool {
	ret := &protocol.Tool{
		Name:        operation.OperationID,
		Description: operation.Description,
		InputSchema: protocol.InputSchema{
			Type:       protocol.Object,
			Properties: make(map[string]*protocol.Property),
		},
	}
	// 处理description，保证非空
	if ret.Description == "" {
		if operation.Summary != "" {
			ret.Description = operation.Summary
		} else {
			ret.Description = operation.OperationID
		}
	}
	// 解析路径参数、查询参数、header 参数等
	if operation.Parameters != nil {
		properties, requireds := Parameters2MCPProtocolProperties(operation.Parameters)
		for field, property := range properties {
			ret.InputSchema.Properties[field] = property
		}
		ret.InputSchema.Required = append(ret.InputSchema.Required, requireds...)
	}
	// 解析请求体
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		for _, mediaType := range operation.RequestBody.Value.Content {
			if mediaType.Schema != nil && mediaType.Schema.Value != nil {
				for field, property := range Schemas2MCPProtocolProperties(mediaType.Schema.Value.Properties) {
					ret.InputSchema.Properties[field] = property
				}
				ret.InputSchema.Required = append(ret.InputSchema.Required, mediaType.Schema.Value.Required...)
			}
		}
	}
	return ret
}

func Parameters2MCPProtocolProperties(parameters openapi3.Parameters) (map[string]*protocol.Property, []string) {
	if parameters == nil {
		return nil, nil
	}

	rets := make(map[string]*protocol.Property)
	var requireds []string
	for _, parameter := range parameters {
		if parameter.Value == nil {
			continue
		}
		field := parameter.Value.In + "-" + parameter.Value.Name
		rets[field] = Parameter2MCPProtocolProperty(parameter.Value)
		if parameter.Value.Required {
			requireds = append(requireds, field)
		}
	}

	return rets, requireds
}

func Parameter2MCPProtocolProperty(parameter *openapi3.Parameter) *protocol.Property {
	if parameter == nil {
		return nil
	}

	dataType := ParameterType2MCPProtocolDataType(parameter)
	ret := &protocol.Property{
		Type:        protocol.PropertyType{dataType},
		Description: parameter.Description,
		Required:    parameter.Schema.Value.Required,
		// todo enum
	}
	switch dataType {
	case protocol.ObjectT:
		if parameter.Schema != nil && parameter.Schema.Value != nil {
			ret.Properties = Schemas2MCPProtocolProperties(parameter.Schema.Value.Properties)
		}
	case protocol.Array:
		if parameter.Schema != nil && parameter.Schema.Value != nil && parameter.Schema.Value.Items != nil {
			ret.Items = Schema2MCPProtocolProperty(parameter.Schema.Value.Items.Value)
		}
	default:
	}

	return ret
}

func Schemas2MCPProtocolProperties(schemas openapi3.Schemas) map[string]*protocol.Property {
	if schemas == nil {
		return nil
	}

	rets := make(map[string]*protocol.Property)
	for propName, propSchema := range schemas {
		if propSchema == nil || propSchema.Value == nil {
			continue
		}
		rets[propName] = Schema2MCPProtocolProperty(propSchema.Value)
	}

	return rets
}

func Schema2MCPProtocolProperty(schema *openapi3.Schema) *protocol.Property {
	if schema == nil {
		return nil
	}

	dataType := SchemaType2MCPProtocolDataType(schema)
	ret := &protocol.Property{
		Type:        protocol.PropertyType{dataType},
		Description: schema.Description,
		Required:    schema.Required,
		// todo enum
	}
	switch dataType {
	case protocol.ObjectT:
		ret.Properties = Schemas2MCPProtocolProperties(schema.Properties)
	case protocol.Array:
		if schema.Items != nil {
			ret.Items = Schema2MCPProtocolProperty(schema.Items.Value)
		}
	default:
	}

	return ret
}

// ParameterType2MCPProtocolDataType 获取参数类型
func ParameterType2MCPProtocolDataType(parameter *openapi3.Parameter) protocol.DataType {
	if parameter.Schema == nil {
		return protocol.Null
	}
	return SchemaType2MCPProtocolDataType(parameter.Schema.Value)
}

// SchemaType2MCPProtocolDataType 获取 schema 的类型
func SchemaType2MCPProtocolDataType(schema *openapi3.Schema) protocol.DataType {
	if schema == nil {
		return protocol.Null
	}
	switch schema.Type {
	case openapi3.TypeObject:
		return protocol.ObjectT
	case openapi3.TypeArray:
		return protocol.Array
	case openapi3.TypeString:
		return protocol.String
	case openapi3.TypeNumber:
		return protocol.Number
	case openapi3.TypeInteger:
		return protocol.Integer
	case openapi3.TypeBoolean:
		return protocol.Boolean
	default:
		return protocol.Null
	}
}
