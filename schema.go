// Copyright 2022, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/pulumi/pulumi-go-provider/internal/introspect"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	STRING  = "string"
	INT     = "integer"
	BOOL    = "boolean"
	FLOAT   = "number"
	ARRAY   = "array"
	MAP     = "object"
	OBJECT  = "object"
	UNKNOWN = "unknown"
)

type serializationInfo struct {
	pkgname   string
	resources map[reflect.Type]string
	types     map[reflect.Type]string
	inputMap  inputToImplementor
}

// Serialize a package to JSON Schema.
func serialize(opts options) (string, error) {
	pkgSpec, err := serializeSchema(opts)
	if err != nil {
		return "", err
	}

	schemaJSON, err := json.MarshalIndent(pkgSpec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(schemaJSON), nil
}

// Get the packagespec given resources, etc.
func serializeSchema(opts options) (schema.PackageSpec, error) {
	spec := schema.PackageSpec{}
	spec.Resources = make(map[string]schema.ResourceSpec)
	spec.Types = make(map[string]schema.ComplexTypeSpec)
	spec.Name = opts.Name
	spec.Version = opts.Version.String()

	info := serializationInfo{}
	info.pkgname = opts.Name
	info.resources = make(map[reflect.Type]string)
	info.types = make(map[reflect.Type]string)
	info.inputMap = initializeInputMap()

	for _, resource := range opts.Resources {
		t := reflect.TypeOf(resource)
		t = dereference(t)
		tokenType, err := introspect.GetToken(tokens.Package(info.pkgname), resource)
		if err != nil {
			return schema.PackageSpec{}, err
		}
		token := tokenType.String()
		info.resources[t] = token
	}

	for _, typ := range opts.Types {
		t := reflect.TypeOf(typ)
		t = dereference(t)
		tokenType, err := introspect.GetToken(tokens.Package(info.pkgname), typ)
		if err != nil {
			return schema.PackageSpec{}, err
		}
		token := tokenType.String()
		info.types[t] = token
	}
	for _, component := range opts.Components {
		t := reflect.TypeOf(component)
		t = dereference(t)
		tokenType, err := introspect.GetToken(tokens.Package(info.pkgname), component)
		if err != nil {
			return schema.PackageSpec{}, err
		}
		token := tokenType.String()
		info.resources[t] = token
	}

	for _, resource := range opts.Resources {
		resourceSpec, err := serializeResource(resource, info)
		if err != nil {
			return schema.PackageSpec{}, err
		}
		token := info.resources[dereference(reflect.TypeOf(resource))]
		spec.Resources[token] = resourceSpec
	}
	//Components are essentially resources, I don't believe they are differentiated in the schema
	for _, component := range opts.Components {
		componentSpec, err := serializeResource(component, info)
		if err != nil {
			return schema.PackageSpec{}, err
		}
		componentSpec.IsComponent = true
		token := info.resources[dereference(reflect.TypeOf(component))]
		spec.Resources[token] = componentSpec
	}

	for _, t := range opts.Types {
		typeSpec, err := serializeType(t, info)
		if err != nil {
			return schema.PackageSpec{}, err
		}
		token := info.types[dereference(reflect.TypeOf(t))]
		spec.Types[token] = typeSpec
	}
	over := opts.PartialSpec
	merged, err := merge(spec, over)
	spec = merged.(schema.PackageSpec)
	if err != nil {
		return schema.PackageSpec{}, err
	}
	return spec, nil
}

func initializeInputMap() inputToImplementor {
	var inputMap inputToImplementor = make(map[reflect.Type]reflect.Type)
	//IntInput to int
	inputMap.add((*pulumi.IntInput)(nil), (*int)(nil))

	//IntPtrInput to *int
	inputMap.add((*pulumi.IntPtrInput)(nil), (**int)(nil))

	//IntArrayInput to []int
	inputMap.add((*pulumi.IntArrayInput)(nil), (*[]int)(nil))

	//IntMapInput to map[string]int
	inputMap.add((*pulumi.IntMapInput)(nil), (*map[string]int)(nil))

	//IntArrayMapInput to map[string][]int
	inputMap.add((*pulumi.IntArrayMapInput)(nil), (*map[string][]int)(nil))

	//IntMapArrayInput to []map[string]int
	inputMap.add((*pulumi.IntMapArrayInput)(nil), (*[]map[string]int)(nil))

	//IntMapMapInput to map[string]map[string]int
	inputMap.add((*pulumi.IntMapMapInput)(nil), (*map[string]map[string]int)(nil))

	//IntArrayArrayInput to [][]int
	inputMap.add((*pulumi.IntArrayArrayInput)(nil), (*[][]int)(nil))

	//StringInput to string
	inputMap.add((*pulumi.StringInput)(nil), (*string)(nil))

	//StringPtrInput to *string
	inputMap.add((*pulumi.StringPtrInput)(nil), (**string)(nil))

	//StringArrayInput to []string
	inputMap.add((*pulumi.StringArrayInput)(nil), (*[]string)(nil))

	//StringMapInput to map[string]string
	inputMap.add((*pulumi.StringMapInput)(nil), (*map[string]string)(nil))

	//StringArrayMapInput to map[string][]string
	inputMap.add((*pulumi.StringArrayMapInput)(nil), (*map[string][]string)(nil))

	//StringMapArrayInput to []map[string]string
	inputMap.add((*pulumi.StringMapArrayInput)(nil), (*[]map[string]string)(nil))

	//StringMapMapInput to map[string]map[string]string
	inputMap.add((*pulumi.StringMapMapInput)(nil), (*map[string]map[string]string)(nil))

	//StringArrayArrayInput to [][]string
	inputMap.add((*pulumi.StringArrayArrayInput)(nil), (*[][]string)(nil))

	//URNInput to pulumi.URN
	inputMap.add((*pulumi.URNInput)(nil), (*pulumi.URN)(nil))

	//URNPtrInput to *pulumi.URN
	inputMap.add((*pulumi.URNPtrInput)(nil), (**pulumi.URN)(nil))

	//URNArrayInput to []pulumi.URN
	inputMap.add((*pulumi.URNArrayInput)(nil), (*[]pulumi.URN)(nil))

	//URNMapInput to map[string]pulumi.URN
	inputMap.add((*pulumi.URNMapInput)(nil), (*map[string]pulumi.URN)(nil))

	//URNArrayMapInput to map[string][]pulumi.URN
	inputMap.add((*pulumi.URNArrayMapInput)(nil), (*map[string][]pulumi.URN)(nil))

	//URNMapArrayInput to []map[string]pulumi.URN
	inputMap.add((*pulumi.URNMapArrayInput)(nil), (*[]map[string]pulumi.URN)(nil))

	//URNMapMapInput to map[string]map[string]pulumi.URN
	inputMap.add((*pulumi.URNMapMapInput)(nil), (*map[string]map[string]pulumi.URN)(nil))

	//URNArrayArrayInput to [][]pulumi.URN
	inputMap.add((*pulumi.URNArrayArrayInput)(nil), (*[][]pulumi.URN)(nil))

	//ArchiveInput to pulumi.Archive
	inputMap.add((*pulumi.ArchiveInput)(nil), (*pulumi.Archive)(nil))

	//ArchiveArrayInput to []pulumi.Archive
	inputMap.add((*pulumi.ArchiveArrayInput)(nil), (*[]pulumi.Archive)(nil))

	//ArchiveMapInput to map[string]pulumi.Archive
	inputMap.add((*pulumi.ArchiveMapInput)(nil), (*map[string]pulumi.Archive)(nil))

	//ArchiveArrayMapInput to map[string][]pulumi.Archive
	inputMap.add((*pulumi.ArchiveArrayMapInput)(nil), (*map[string][]pulumi.Archive)(nil))

	//ArchiveMapArrayInput to []map[string]pulumi.Archive
	inputMap.add((*pulumi.ArchiveMapArrayInput)(nil), (*[]map[string]pulumi.Archive)(nil))

	//ArchiveMapMapInput to map[string]map[string]pulumi.Archive
	inputMap.add((*pulumi.ArchiveMapMapInput)(nil), (*map[string]map[string]pulumi.Archive)(nil))

	//ArchiveArrayArrayInput to [][]pulumi.Archive
	inputMap.add((*pulumi.ArchiveArrayArrayInput)(nil), (*[][]pulumi.Archive)(nil))

	//AssetInput to pulumi.Asset
	inputMap.add((*pulumi.AssetInput)(nil), (*pulumi.Asset)(nil))

	//AssetArrayInput to []pulumi.Asset
	inputMap.add((*pulumi.AssetArrayInput)(nil), (*[]pulumi.Asset)(nil))

	//AssetMapInput to map[string]pulumi.Asset
	inputMap.add((*pulumi.AssetMapInput)(nil), (*map[string]pulumi.Asset)(nil))

	//AssetArrayMapInput to map[string][]pulumi.Asset
	inputMap.add((*pulumi.AssetArrayMapInput)(nil), (*map[string][]pulumi.Asset)(nil))

	//AssetMapArrayInput to []map[string]pulumi.Asset
	inputMap.add((*pulumi.AssetMapArrayInput)(nil), (*[]map[string]pulumi.Asset)(nil))

	//AssetMapMapInput to map[string]map[string]pulumi.Asset
	inputMap.add((*pulumi.AssetMapMapInput)(nil), (*map[string]map[string]pulumi.Asset)(nil))

	//AssetArrayArrayInput to [][]pulumi.Asset
	inputMap.add((*pulumi.AssetArrayArrayInput)(nil), (*[][]pulumi.Asset)(nil))

	//AssetOrArchiveInput to pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveInput)(nil), (*pulumi.AssetOrArchive)(nil))

	//AssetOrArchiveArrayInput to []pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveArrayInput)(nil), (*[]pulumi.AssetOrArchive)(nil))

	//AssetOrArchiveMapInput to map[string]pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveMapInput)(nil), (*map[string]pulumi.AssetOrArchive)(nil))

	//AssetOrArchiveArrayMapInput to map[string][]pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveArrayMapInput)(nil), (*map[string][]pulumi.AssetOrArchive)(nil))

	//AssetOrArchiveMapArrayInput to []map[string]pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveMapArrayInput)(nil), (*[]map[string]pulumi.AssetOrArchive)(nil))

	//AssetOrArchiveMapMapInput to map[string]map[string]pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveMapMapInput)(nil), (*map[string]map[string]pulumi.AssetOrArchive)(nil))

	//AssetOrArchiveArrayArrayInput to [][]pulumi.AssetOrArchive
	inputMap.add((*pulumi.AssetOrArchiveArrayArrayInput)(nil), (*[][]pulumi.AssetOrArchive)(nil))

	//BoolInput to bool
	inputMap.add((*pulumi.BoolInput)(nil), (*bool)(nil))

	//BoolArrayInput to []bool
	inputMap.add((*pulumi.BoolArrayInput)(nil), (*[]bool)(nil))

	//BoolMapInput to map[string]bool
	inputMap.add((*pulumi.BoolMapInput)(nil), (*map[string]bool)(nil))

	//BoolArrayMapInput to map[string][]bool
	inputMap.add((*pulumi.BoolArrayMapInput)(nil), (*map[string][]bool)(nil))

	//BoolMapArrayInput to []map[string]bool
	inputMap.add((*pulumi.BoolMapArrayInput)(nil), (*[]map[string]bool)(nil))

	//BoolMapMapInput to map[string]map[string]bool
	inputMap.add((*pulumi.BoolMapMapInput)(nil), (*map[string]map[string]bool)(nil))

	//BoolArrayArrayInput to [][]bool
	inputMap.add((*pulumi.BoolArrayArrayInput)(nil), (*[][]bool)(nil))

	//IDInput to pulumi.ID
	inputMap.add((*pulumi.IDInput)(nil), (*pulumi.ID)(nil))

	//IDPtrInput to *pulumi.ID
	inputMap.add((*pulumi.IDPtrInput)(nil), (**pulumi.ID)(nil))

	//IDArrayInput to []pulumi.ID
	inputMap.add((*pulumi.IDArrayInput)(nil), (*[]pulumi.ID)(nil))

	//IDMapInput to map[string]pulumi.ID
	inputMap.add((*pulumi.IDMapInput)(nil), (*map[string]pulumi.ID)(nil))

	//IDArrayMapInput to map[string][]pulumi.ID
	inputMap.add((*pulumi.IDArrayMapInput)(nil), (*map[string][]pulumi.ID)(nil))

	//IDMapArrayInput to []map[string]pulumi.ID
	inputMap.add((*pulumi.IDMapArrayInput)(nil), (*[]map[string]pulumi.ID)(nil))

	//IDMapMapInput to map[string]map[string]pulumi.ID
	inputMap.add((*pulumi.IDMapMapInput)(nil), (*map[string]map[string]pulumi.ID)(nil))

	//IDArrayArrayInput to [][]pulumi.ID
	inputMap.add((*pulumi.IDArrayArrayInput)(nil), (*[][]pulumi.ID)(nil))

	//ArrayInput to []any
	inputMap.add((*pulumi.ArrayInput)(nil), (*[]any)(nil))

	//MapInput to map[string]any
	inputMap.add((*pulumi.MapInput)(nil), (*map[string]any)(nil))

	//ArrayMapInput to map[string][]any
	inputMap.add((*pulumi.ArrayMapInput)(nil), (*map[string][]any)(nil))

	//MapArrayInput to []map[string]any
	inputMap.add((*pulumi.MapArrayInput)(nil), (*[]map[string]any)(nil))

	//MapMapInput to map[string]map[string]any
	inputMap.add((*pulumi.MapMapInput)(nil), (*map[string]map[string]any)(nil))

	//ArrayArrayInput to [][]any
	inputMap.add((*pulumi.ArrayArrayInput)(nil), (*[][]any)(nil))

	//ArrayArrayMapInput to map[string][][]any
	inputMap.add((*pulumi.ArrayArrayMapInput)(nil), (*map[string][][]any)(nil))

	//Float65Input to float64
	inputMap.add((*pulumi.Float64Input)(nil), (*float64)(nil))

	//Float64PtrInput to *float64
	inputMap.add((*pulumi.Float64PtrInput)(nil), (**float64)(nil))

	//Float64ArrayInput to []float64
	inputMap.add((*pulumi.Float64ArrayInput)(nil), (*[]float64)(nil))

	//Float64MapInput to map[string]float64
	inputMap.add((*pulumi.Float64MapInput)(nil), (*map[string]float64)(nil))

	//Float64ArrayMapInput to map[string][]float64
	inputMap.add((*pulumi.Float64ArrayMapInput)(nil), (*map[string][]float64)(nil))

	//Float64MapArrayInput to []map[string]float64
	inputMap.add((*pulumi.Float64MapArrayInput)(nil), (*[]map[string]float64)(nil))

	//Float64MapMapInput to map[string]map[string]float64
	inputMap.add((*pulumi.Float64MapMapInput)(nil), (*map[string]map[string]float64)(nil))

	//Float64ArrayArrayInput to [][]float64
	inputMap.add((*pulumi.Float64ArrayArrayInput)(nil), (*[][]float64)(nil))

	//ResourceInput to pulumi.Resource
	inputMap.add((*pulumi.ResourceInput)(nil), (*pulumi.Resource)(nil))

	//ResourceArrayInput to []pulumi.Resource
	inputMap.add((*pulumi.ResourceArrayInput)(nil), (*[]pulumi.Resource)(nil))

	return inputMap
}

type inputToImplementor map[reflect.Type]reflect.Type

func (m inputToImplementor) add(k any, v any) {
	m[reflect.TypeOf(k).Elem()] = reflect.TypeOf(v).Elem()
}

func merge(spec, over any) (any, error) {
	//iterate over fields in the overriding spec
	t := reflect.TypeOf(over)
	sv := reflect.ValueOf(spec)
	ov := reflect.ValueOf(over)
	for i := 0; i < t.NumField(); i++ {
		//for primitives
		if t.Field(i).Type.Kind() == reflect.Struct || t.Field(i).Type.Kind() == reflect.Interface {
			merged, err := merge(sv.Field(i).Interface(), ov.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			sv.Field(i).Set(reflect.ValueOf(merged))

		} else if t.Field(i).Type.Kind() == reflect.Array {
			if t.Field(i).Type.Elem().Kind() == reflect.Struct || t.Field(i).Type.Elem().Kind() == reflect.Interface {
				merged, err := mergeStructArraysByKey[any,
					string](sv.Field(i).Interface().([]any), ov.Field(i).Interface().([]any), "Name")
				if err != nil {
					return nil, err
				}
				sv.Field(i).Set(reflect.ValueOf(merged))
			}
		} else if t.Field(i).Type.Kind() == reflect.Map {
			if t.Field(i).Type.Key().Kind() != reflect.String {
				return nil, fmt.Errorf("map key type must be string")
			}
			merged := mergeMapsOverride[any](sv.Field(i).Interface().(map[string]any), ov.Field(i).Interface().(map[string]any))
			sv.Field(i).Set(reflect.ValueOf(merged))
		} else {
			if !ov.Field(i).IsNil() {
				sv.Field(i).Set(ov.Field(i))
			}
		}
	}
	return sv.Interface().(struct{}), nil
}

// Get the resourceSpec for a single resource
func serializeResource(rawResource any, info serializationInfo) (schema.ResourceSpec, error) {
	v := reflect.ValueOf(rawResource)
	for v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	properties := make(map[string]schema.PropertySpec)
	inputProperties := make(map[string]schema.PropertySpec)
	requiredInputs := make([]string, 0)
	required := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i)
		fieldType := field.Type
		vField := v.Field(i)

		tags, err := introspect.ParseTag(field)
		if err != nil {
			return schema.ResourceSpec{}, err
		}
		if tags.Internal {
			continue
		}

		isInput := field.Type.Implements(reflect.TypeOf(new(pulumi.Input)).Elem())
		_, isOutput := vField.Interface().(pulumi.Output)

		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if isOutput {
			fieldType = reflect.New(fieldType).Elem().Interface().(pulumi.Output).ElementType()
		} else if isInput {
			if _, ok := info.inputMap[fieldType]; !ok {
				return schema.ResourceSpec{}, fmt.Errorf("input %s for property"+
					"%s has type %s, which is not a valid input type", field.Name, t, fieldType)
			}
			fieldType = info.inputMap[fieldType]
		}
		fieldType = dereference(fieldType)
		serialized, err := serializeProperty(fieldType, tags.Description, info)
		if err != nil {
			return schema.ResourceSpec{}, err
		}
		if ((!tags.Output) || isInput) && !isOutput {
			inputProperties[tags.Name] = serialized
			if !tags.Optional {
				requiredInputs = append(requiredInputs, tags.Name)
			}
		}
		properties[tags.Name] = serialized
		if !tags.Optional {
			required = append(required, tags.Name)
		}
	}

	spec := schema.ResourceSpec{}
	spec.ObjectTypeSpec.Properties = properties
	spec.InputProperties = inputProperties
	spec.RequiredInputs = requiredInputs
	spec.Required = required
	return spec, nil
}

//Get the propertySpec for a single property
func serializeProperty(t reflect.Type, description string, info serializationInfo) (schema.PropertySpec, error) {
	typeName := getTypeName(t)
	if isTypeOrResource(t, info) {
		typeSpec, err := serializeRef(t, info)
		if err != nil {
			return schema.PropertySpec{}, err
		}
		return schema.PropertySpec{
			Description: description,
			TypeSpec:    *typeSpec,
		}, nil
	} else if typeName == ARRAY {
		itemSpec, err := serializeTypeRef(t.Elem(), info)
		if err != nil {
			return schema.PropertySpec{}, err
		}
		return schema.PropertySpec{
			Description: description,
			TypeSpec: schema.TypeSpec{
				Type:  typeName,
				Items: itemSpec,
			},
		}, nil
	} else if typeName == MAP {
		valSpec, err := serializeTypeRef(t.Elem(), info)
		if err != nil {
			return schema.PropertySpec{}, err
		}
		if t.Key().Kind() != reflect.String {
			return schema.PropertySpec{}, fmt.Errorf("map keys must be strings")
		}
		return schema.PropertySpec{
			Description: description,
			TypeSpec: schema.TypeSpec{
				Type:                 "object", //There is no map type in the schema
				AdditionalProperties: valSpec,
			},
		}, nil
	} else if typeName != UNKNOWN {
		return schema.PropertySpec{
			Description: description,
			TypeSpec: schema.TypeSpec{
				Type: typeName,
			},
		}, nil
	} else {
		return schema.PropertySpec{}, fmt.Errorf("unknown type %s", t)
	}
}

func serializeRef(t reflect.Type, info serializationInfo) (*schema.TypeSpec, error) {
	t = dereference(t)
	if token, ok := info.resources[t]; ok {
		return &schema.TypeSpec{
			Type: "#/resources/" + token,
		}, nil
	}
	if token, ok := info.types[t]; ok {
		return &schema.TypeSpec{
			Type: "#/types/" + token,
		}, nil
	}
	return nil, fmt.Errorf("unknown type %s", t)

}

func isTypeOrResource(t reflect.Type, info serializationInfo) bool {
	t = dereference(t)
	_, isResource := info.resources[t]
	if isResource {
		return true
	}
	_, isType := info.types[t]
	return isType
}

//Get the typeSpec for a single type
func getTypeName(t reflect.Type) string {
	var typeName string
	switch t.Kind() {
	case reflect.String:
		typeName = STRING
	case reflect.Bool:
		typeName = BOOL
	case reflect.Int:
		typeName = INT
	case reflect.Float64:
		typeName = FLOAT
	case reflect.Array, reflect.Slice:
		typeName = ARRAY
	case reflect.Map:
		typeName = MAP
	case reflect.Interface, reflect.Struct: //Should maps be objects?
		typeName = OBJECT
	default:
		typeName = UNKNOWN
	}
	return typeName
}

func serializeTypeRef(t reflect.Type, info serializationInfo) (*schema.TypeSpec, error) {
	typeName := getTypeName(t)
	if isTypeOrResource(t, info) {
		return serializeRef(t, info)
	}
	if typeName == ARRAY {
		itemSpec, err := serializeTypeRef(t.Elem(), info)
		if err != nil {
			return nil, err
		}
		return &schema.TypeSpec{
			Type:  typeName,
			Items: itemSpec,
		}, nil
	} else if typeName == MAP {
		valSpec, err := serializeTypeRef(t.Elem(), info)
		if err != nil {
			return nil, err
		}
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map keys must be strings")
		}
		return &schema.TypeSpec{
			Type:                 "object", //There is no map type in the schema
			AdditionalProperties: valSpec,
		}, nil
	} else if typeName != UNKNOWN {
		return &schema.TypeSpec{
			Type: typeName,
		}, nil
	} else {
		return &schema.TypeSpec{}, fmt.Errorf("unknown type %s", t)
	}
}

func serializeType(typ any, info serializationInfo) (schema.ComplexTypeSpec, error) {
	t := reflect.TypeOf(typ)
	t = dereference(t)
	typeName := getTypeName(t)
	if typeName == "object" {
		properties := make(map[string]schema.PropertySpec)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tags, err := introspect.ParseTag(field)
			if err != nil {
				return schema.ComplexTypeSpec{}, err
			}
			properties[tags.Name], err = serializeProperty(field.Type, tags.Description, info)
			if err != nil {
				return schema.ComplexTypeSpec{}, err
			}
		}
		return schema.ComplexTypeSpec{
			ObjectTypeSpec: schema.ObjectTypeSpec{
				Type:       "object",
				Properties: properties,
			},
		}, nil
	}
	enumVals := make([]schema.EnumValueSpec, 0)

	fmt.Fprintf(os.Stderr, "WARNING! Enum types must be manually specified"+
		"Overwrite the autogenerated type %s with your own.", t)
	return schema.ComplexTypeSpec{
		ObjectTypeSpec: schema.ObjectTypeSpec{
			Type: typeName,
		},
		Enum: enumVals,
	}, nil
}

func mergePrimitiveArrays(base, override []any) []any {
	m := make(map[any]bool)
	for _, x := range base {
		m[x] = true
	}
	//If an element in override is not in base, append it
	for _, y := range override {
		if !m[y] {
			base = append(base, y)
		}
	}
	return base
}

//Merge two arrays of structs which have the string property "Name" by their names
func mergeStructArraysByKey[T any, K comparable](base, override []T, fieldName string) ([]T, error) {
	m := make(map[K]T)
	//Check that type T has field fieldName
	t := reflect.TypeOf((*T)(nil)).Elem()
	field, ok := t.FieldByName(fieldName)
	if !ok {
		return nil, fmt.Errorf("type %s does not have field %s", t, fieldName)
	}
	//Check that field fieldName is of type K
	k := reflect.TypeOf((*K)(nil)).Elem()
	if field.Type != k {
		return nil, fmt.Errorf("type %s field %s is not of type %s", t, fieldName, k)
	}

	for _, x := range base {
		key := reflect.ValueOf(x).FieldByName(fieldName).Interface().(K)
		m[key] = x
	}

	for _, y := range override {
		key := reflect.ValueOf(y).FieldByName(fieldName).Interface().(K)
		if _, ok := m[key]; !ok {
			base = append(base, y)
		} //We could overwrite or we could call merge(), not sure which is desired behavior
	}

	return base, nil
}

func mergeMapsOverride[T any](base, override map[string]T) map[string]T {
	for k, v := range override {
		base[k] = v
	}
	return base
}

func dereference(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
