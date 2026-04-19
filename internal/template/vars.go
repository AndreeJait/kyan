package template

import (
	"fmt"
	"strings"
)

type ModuleVars struct {
	ModuleName      string  // e.g., "Product"
	ModuleNameLower string  // e.g., "product"
	ModulePath      string  // e.g., "github.com/user/project"
	Fields          []Field
	WithCaching     bool
	WithAuth        bool
}

type Field struct {
	Name           string // e.g., "Title"
	NameLower      string // e.g., "title"
	GoType         string // e.g., "string", "int", "float64", "bool", "time.Time"
	GormTag        string // e.g., `gorm:"not null"`
	SQLType        string // e.g., "VARCHAR(255)", "TEXT", "INTEGER"
	SQLDefault     string // e.g., "DEFAULT FALSE", "DEFAULT NOW()", ""
	NeedsDatatypes bool   // true if GoType is datatypes.JSON
}

type FieldTypeInfo struct {
	GoType   string
	SQLType  string
	SQLDefault string
	NeedsDatatypes bool
}

var FieldTypeMap = map[string]FieldTypeInfo{
	"string": {GoType: "string", SQLType: "VARCHAR(255)", SQLDefault: "", NeedsDatatypes: false},
	"text":   {GoType: "string", SQLType: "TEXT", SQLDefault: "", NeedsDatatypes: false},
	"int":    {GoType: "int", SQLType: "INTEGER", SQLDefault: "", NeedsDatatypes: false},
	"float":  {GoType: "float64", SQLType: "FLOAT", SQLDefault: "", NeedsDatatypes: false},
	"bool":   {GoType: "bool", SQLType: "BOOLEAN", SQLDefault: "DEFAULT FALSE", NeedsDatatypes: false},
	"uuid":   {GoType: "string", SQLType: "VARCHAR(36)", SQLDefault: "", NeedsDatatypes: false},
	"time":   {GoType: "time.Time", SQLType: "TIMESTAMP WITH TIME ZONE", SQLDefault: "DEFAULT NOW()", NeedsDatatypes: false},
	"json":   {GoType: "datatypes.JSON", SQLType: "JSONB", SQLDefault: "", NeedsDatatypes: true},
}

func ParseFields(fieldsStr string) ([]Field, error) {
	if fieldsStr == "" {
		return nil, nil
	}

	var fields []Field
	parts := strings.Fields(fieldsStr)

	for _, part := range parts {
		nameType := strings.SplitN(part, ":", 2)
		if len(nameType) != 2 {
			return nil, fmt.Errorf("invalid field format: %q (expected name:type)", part)
		}

		name, typeKey := nameType[0], nameType[1]
		info, ok := FieldTypeMap[typeKey]
		if !ok {
			return nil, fmt.Errorf("unknown field type: %q (valid: string, text, int, float, bool, uuid, time, json)", typeKey)
		}

		gormTag := ""
		switch typeKey {
		case "string":
			gormTag = `gorm:"not null"`
		case "bool":
			gormTag = `gorm:"default:false"`
		default:
			gormTag = ""
		}

		fields = append(fields, Field{
			Name:           strings.Title(name),
			NameLower:      name,
			GoType:         info.GoType,
			GormTag:        gormTag,
			SQLType:        info.SQLType,
			SQLDefault:     info.SQLDefault,
			NeedsDatatypes: info.NeedsDatatypes,
		})
	}

	return fields, nil
}

func (m *ModuleVars) NeedsDatatypesImport() bool {
	for _, f := range m.Fields {
		if f.NeedsDatatypes {
			return true
		}
	}
	return false
}