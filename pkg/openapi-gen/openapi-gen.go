// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package openapi_gen

import (
	"context"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	AdditionalPropertyUnchanged    = "AdditionalPropertyUnchanged"
	AdditionalPropertiesUnchTarget = "AdditionalPropertiesUnchTarget"
)

var swagger openapi3.Swagger

var respGet200Desc = "GET OK 200"
var pathPrefix string
var targetParameter *openapi3.ParameterRef

type ApiGenSettings struct {
	ModelType    string
	ModelVersion string
	Title        string
	Description  string
	TargetAlias  string
}

type pathType uint8

const (
	Undefined pathType = iota
	pathTypeListMultiple
	pathTypeContainer
)

func (pt pathType) string() string {
	switch pt {
	case pathTypeListMultiple:
		return "List"
	case pathTypeContainer:
		return "Container"
	default:
		return "undefined"
	}
}

func (settings *ApiGenSettings) ApplyDefaults() {
	if settings.ModelType == "" {
		panic("ModelType not specified")
	}

	// Fill in defaults for any unset settings
	if settings.ModelVersion == "" {
		settings.ModelVersion = "0.0.1"
	}
	if settings.Title == "" {
		settings.Title = fmt.Sprintf("%s onos-config model plugin", settings.ModelType)
	}
	if settings.Description == "" {
		settings.Description = fmt.Sprintf("OpenAPI 3 specification is generated from "+
			"%s onos-config model plugin", settings.ModelType)
	}
	if settings.TargetAlias == "" {
		settings.TargetAlias = "target"
	}
}

func BuildOpenapi(yangSchema *ytypes.Schema, settings *ApiGenSettings) (*openapi3.Swagger, error) {
	settings.ApplyDefaults()

	pathPrefix = fmt.Sprintf("/%s/v%s/{%s}", strings.ToLower(settings.ModelType), settings.ModelVersion, settings.TargetAlias)
	targetParameter = targetParam(settings.TargetAlias)

	topEntry := yangSchema.SchemaTree["Device"]
	paths, components, err := buildSchema(topEntry, yang.TSFalse, "", settings.TargetAlias)
	if err != nil {
		return nil, err
	}

	components.Parameters = make(map[string]*openapi3.ParameterRef)
	components.Parameters[settings.TargetAlias] = targetParameter

	// At the root of the API, add in the definition of "additionalPropertyTarget"
	schemaValTarget := openapi3.NewObjectSchema()
	schemaValTarget.Title = settings.TargetAlias
	schemaValTarget.Type = "string"
	schemaValTarget.Description = fmt.Sprintf("an override of the %s (target)", settings.TargetAlias)

	schemaValAddTarget := openapi3.NewObjectSchema()
	schemaValAddTarget.Properties = make(map[string]*openapi3.SchemaRef)
	schemaValAddTarget.Title = additionalPropertyTarget(settings.TargetAlias)
	schemaValAddTarget.Description = fmt.Sprintf("Optionally specify a %s other than the default (only on PATCH method)", settings.TargetAlias)
	schemaValAddTargetRef := schemaValAddTarget.NewRef()
	schemaValAddTarget.Properties[settings.TargetAlias] = schemaValTarget.NewRef()
	components.Schemas[additionalPropertyTarget(settings.TargetAlias)] = schemaValAddTargetRef

	schemaValUnchanged := openapi3.NewObjectSchema()
	schemaValUnchanged.Title = "unchanged"
	schemaValUnchanged.Type = "string"
	schemaValUnchanged.Description = "A comma seperated list of unchanged mandatory attribute names"

	schemaValAddUnchanged := openapi3.NewObjectSchema()
	schemaValAddUnchanged.Properties = make(map[string]*openapi3.SchemaRef)
	schemaValAddUnchanged.Title = AdditionalPropertyUnchanged
	schemaValAddUnchanged.Description = "To optionally omit 'required' properties, add them to 'unchanged' list"

	schemaValAddUnchanged.Properties["unchanged"] = schemaValUnchanged.NewRef()
	components.Schemas[AdditionalPropertyUnchanged] = schemaValAddUnchanged.NewRef()

	schemaValAddBoth := openapi3.NewObjectSchema()
	schemaValAddBoth.Properties = make(map[string]*openapi3.SchemaRef)
	schemaValAddBoth.Title = AdditionalPropertiesUnchTarget
	schemaValAddBoth.Description = fmt.Sprintf("both the additional property 'unchanged' and the '%s'", settings.TargetAlias)
	schemaValAddBoth.Properties["unchanged"] = schemaValUnchanged.NewRef()
	schemaValAddBoth.Properties[settings.TargetAlias] = schemaValTarget.NewRef()
	components.Schemas[AdditionalPropertiesUnchTarget] = schemaValAddBoth.NewRef()

	swagger = openapi3.Swagger{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   settings.Title,
			Version: settings.ModelVersion,
			Contact: &openapi3.Contact{
				Name:  "Open Networking Foundation",
				URL:   "https://opennetworking.org",
				Email: "info@opennetworking.org",
			},
			License: &openapi3.License{
				Name: "Apache-2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0",
			},
			Description: settings.Description,
		},
		Paths:      paths,
		Components: *components,
	}

	if err := swagger.Validate(context.Background()); err != nil {
		return nil, err
	}

	swaggerLdr := openapi3.NewSwaggerLoader()
	if err = swaggerLdr.ResolveRefsIn(&swagger, nil); err != nil {
		fmt.Fprintf(os.Stderr, "error on Resolving Refs %v\n", err)
	}

	return &swagger, nil
}

func targetParam(targetAlias string) *openapi3.ParameterRef {

	stringContent := openapi3.NewContent()
	mt := openapi3.NewMediaType()
	mt.Schema = &openapi3.SchemaRef{
		Value: openapi3.NewStringSchema(),
	}
	stringContent["text/plain; charset=utf-8"] = mt

	targetParam := openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:        targetAlias,
			In:          "path",
			Description: fmt.Sprintf("%s (target in onos-config)", targetAlias),
			Required:    true,
			Content:     stringContent,
		},
	}

	return &targetParam
}

// add AdditionalProperties reference to target to a particular schema
func addAdditionalProperties(schemaVal *openapi3.Schema, name string) {
	if schemaVal.AdditionalProperties != nil {
		name = AdditionalPropertiesUnchTarget
	}
	schemaValAdditionalRef := openapi3.NewObjectSchema()
	schemaValAdditionalRef.Properties = make(map[string]*openapi3.SchemaRef)
	schemaValAdditionalRef.Title = "ref"
	schemaVal.AdditionalProperties = &openapi3.SchemaRef{
		Value: schemaValAdditionalRef,
		Ref:   fmt.Sprintf("#/components/schemas/%s", name),
	}

}

// buildSchema is a recursive function to extract a list of read only paths from a YGOT schema
func buildSchema(deviceEntry *yang.Entry, parentState yang.TriState, parentPath string, targetAlias string) (openapi3.Paths, *openapi3.Components, error) {
	openapiPaths := make(openapi3.Paths)
	openapiComponents := openapi3.Components{
		Schemas:       make(map[string]*openapi3.SchemaRef),
		RequestBodies: make(map[string]*openapi3.RequestBodyRef),
	}

	for _, dirEntry := range deviceEntry.Dir {
		itemPath := fmt.Sprintf("%s/%s", parentPath, dirEntry.Name)
		if dirEntry.IsLeaf() || dirEntry.IsLeafList() {
			// No need to recurse
			var schemaVal *openapi3.Schema
			switch dirEntry.Type.Kind {
			case yang.Ystring:
				schemaVal = openapi3.NewStringSchema()
				if dirEntry.Type.Length != nil {
					min, max, err := yangRange(dirEntry.Type.Length, dirEntry.Type.Kind)
					if err != nil {
						return nil, nil, err
					}
					if min != nil {
						schemaVal.MinLength = uint64(*min)
					}
					if max != nil {
						v := uint64(*max)
						schemaVal.MaxLength = &v
					}
				}
				if dirEntry.Type.Pattern != nil && len(dirEntry.Type.Pattern) > 0 {
					// All we can do is take the first one
					schemaVal.Pattern = dirEntry.Type.Pattern[0]
				}
				if dirEntry.Type.Default != "" {
					schemaVal.Default = dirEntry.Type.Default
				}
			case yang.Yunion:
				schemaVal = openapi3.NewStringSchema()
				if dirEntry.Type.Default != "" {
					schemaVal.Default = dirEntry.Type.Default
				}
			case yang.Yleafref:
				// Lookup type of leafref
				leafRefType := resolveLeafRefType(dirEntry)
				switch leafRefType {
				case yang.Yuint8, yang.Yuint16, yang.Yint8, yang.Yint16:
					schemaVal = openapi3.NewIntegerSchema()
				case yang.Yuint32, yang.Yint32:
					schemaVal = openapi3.NewInt32Schema()
				case yang.Yuint64, yang.Yint64:
					schemaVal = openapi3.NewInt64Schema()
				default:
					schemaVal = openapi3.NewStringSchema()
				}
				if dirEntry.Type.Default != "" {
					schemaVal.Default = dirEntry.Type.Default
				}
				if schemaVal.Extensions == nil {
					schemaVal.Extensions = make(map[string]interface{})
				}
				schemaVal.Extensions["x-leafref"] = dirEntry.Type.Path
			case yang.Yidentityref, yang.Yenum:
				schemaVal = openapi3.NewStringSchema()
				if dirEntry.Type.IdentityBase != nil {
					schemaVal.Enum = make([]interface{}, 0)
					for _, val := range dirEntry.Type.IdentityBase.Values {
						schemaVal.Enum = append(schemaVal.Enum, val.Name)
					}
					sort.Slice(schemaVal.Enum, func(i, j int) bool {
						return schemaVal.Enum[i].(string) < schemaVal.Enum[j].(string)
					})
				}
			case yang.Ybool:
				schemaVal = openapi3.NewBoolSchema()
				// default is now a []string - since this is not a leaf-list there will only be 1 entry
				for _, def := range dirEntry.Default {
					if def == "true" {
						schemaVal.Default = true
						break
					} else if def == "false" {
						schemaVal.Default = false
						break
					}
				}
			case yang.Yuint8, yang.Yuint16, yang.Yint8, yang.Yint16, yang.Yuint32, yang.Yint32, yang.Yuint64, yang.Yint64, yang.Ydecimal64:
				switch dirEntry.Type.Kind {
				case yang.Yuint32, yang.Yint32:
					schemaVal = openapi3.NewInt32Schema()
				case yang.Yuint64, yang.Yint64:
					schemaVal = openapi3.NewInt64Schema()
				case yang.Ydecimal64:
					schemaVal = openapi3.NewFloat64Schema()
				default:
					schemaVal = openapi3.NewIntegerSchema()
				}
				def, err := yangDefault(dirEntry)
				if err != nil {
					return nil, nil, err
				}
				schemaVal.Default = def
				if dirEntry.Type.Range != nil {
					start, end, err := yangRange(dirEntry.Type.Range, dirEntry.Type.Kind)
					if err != nil {
						return nil, nil, err
					}
					if start != nil {
						startFloat := float64(*start)
						schemaVal.Min = &startFloat
					}
					if end != nil {
						endFloat := (float64)(*end)
						schemaVal.Max = &endFloat
					}
				}
			case yang.Ybinary:
				schemaVal = openapi3.NewBytesSchema()
				if dirEntry.Type.Length != nil {
					min, max, err := yangRange(dirEntry.Type.Length, dirEntry.Type.Kind)
					if err != nil {
						return nil, nil, err
					}
					if min != nil {
						schemaVal.MinLength = uint64(*min)
					}
					if max != nil {
						v := uint64(*max)
						schemaVal.MaxLength = &v
					}
				}
				if dirEntry.Type.Default != "" {
					schemaVal.Default = dirEntry.Type.Default
				}
			case yang.Yempty:
				schemaVal = openapi3.NewStringSchema()
				var emptylen uint64 = 0
				schemaVal.MaxLength = &emptylen
			default:
				return nil, nil, fmt.Errorf("unhandled leaf %v %s", dirEntry.Type.Kind, dirEntry.Type.Name)
			}
			schemaVal.Title = dirEntry.Name
			schemaVal.Description = dirEntry.Description
			if dirEntry.Mandatory.Value() {
				schemaVal.Required = append(schemaVal.Required, dirEntry.Name)
			} else if strings.Contains(dirEntry.Parent.Key, dirEntry.Name) {
				schemaVal.Required = append(schemaVal.Required, dirEntry.Name)
			}

			if dirEntry.IsLeaf() {
				openapiComponents.Schemas[toUnderScore(itemPath)] = &openapi3.SchemaRef{
					Value: schemaVal,
				}
			} else { // Leaflist
				arr := openapi3.NewSchema()
				arr.Type = "leaf-list"
				arr.Items = &openapi3.SchemaRef{
					Value: schemaVal,
				}
				arr.Title = dirEntry.Name
				openapiComponents.Schemas[toUnderScore(itemPath)] = &openapi3.SchemaRef{
					Value: arr,
				}
			}
		} else if dirEntry.Kind == yang.ChoiceEntry {
			for name, dir := range dirEntry.Dir {
				_, components, err := buildSchema(dir, dir.Config, parentPath, targetAlias)
				if err != nil {
					return nil, nil, err
				}
				for k, v := range components.Schemas {
					v.Value.Description = fmt.Sprintf("For choice %s:%s", dirEntry.Name, name)
					openapiComponents.Schemas[toUnderScore(k)] = v
				}
			}

		} else if dirEntry.IsContainer() {
			newPath := newPathItem(dirEntry, itemPath, parentPath, pathTypeContainer, targetAlias)
			openapiPaths[pathWithPrefix(itemPath)] = newPath

			paths, components, err := buildSchema(dirEntry, dirEntry.Config, itemPath, targetAlias)
			if err != nil {
				return nil, nil, err
			}
			for k, v := range paths {
				openapiPaths[k] = v
			}

			schemaVal := openapi3.NewObjectSchema()
			schemaVal.Properties = make(map[string]*openapi3.SchemaRef)
			schemaVal.Title = toUnderScore(itemPath)
			schemaVal.Description = dirEntry.Description
			if len(strings.Split(itemPath, "/")) <= 2 {
				addAdditionalProperties(schemaVal, additionalPropertyTarget(targetAlias))
			}
			openapiComponents.Schemas[toUnderScore(itemPath)] = schemaVal.NewRef()

			rbRef := &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().WithContent(
					openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{
						Value: schemaVal,
						Ref:   fmt.Sprintf("#/components/schemas/%s", toUnderScore(itemPath)),
					}),
				),
			}
			openapiComponents.RequestBodies[fmt.Sprintf("RequestBody_%s", toUnderScore(itemPath))] = rbRef

			if newPath.Post != nil && newPath.Post.RequestBody != nil && newPath.Post.RequestBody.Ref != "" {
				newPath.Post.RequestBody.Value = rbRef.Value
			}

			respGet200 := openapi3.NewResponse()
			respGet200.Description = &respGet200Desc
			respGet200.Content = openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{
				Ref:   fmt.Sprintf("#/components/schemas/%s", toUnderscoreWithPathType(itemPath, pathTypeContainer)),
				Value: schemaVal,
			})
			newPath.Get.AddResponse(200, respGet200)

			for k, v := range components.Schemas {
				switch v.Value.Type {
				case "array": // List as a child of container
					schemaPath := pathToSchemaName(itemPath)
					root := k[len(schemaPath) : len(k)-5] // Remove the _List
					if strings.Count(root, "_") == 0 {
						schemaVal.Properties[strings.ToLower(root)] = &openapi3.SchemaRef{
							Ref:   fmt.Sprintf("#/components/schemas/%s", k),
							Value: v.Value,
						}
					}
					openapiComponents.Schemas[k] = v

				case "object": // Container as a child of container
					if _, ok := v.Value.Extensions["x-list-multiple"]; !ok {
						schemaPath := pathToSchemaName(itemPath)
						root := k[len(schemaPath):]
						if v.Value.Title != "" && !strings.Contains(root, "_") {
							schemaVal.Properties[strings.ToLower(lastPartOf(k))] = &openapi3.SchemaRef{
								Ref:   fmt.Sprintf("#/components/schemas/%s", v.Value.Title),
								Value: v.Value,
							}
						}
					}
					openapiComponents.Schemas[k] = v
				case "string", "boolean", "integer", "number": // leaf as a child of list
					if v.Value.Required != nil {
						schemaVal.Required = append(schemaVal.Required, v.Value.Required...)
						sort.Strings(schemaVal.Required)
						v.Value.Required = nil
					}
					schemaVal.Properties[v.Value.Title] = v
				case "leaf-list":
					v.Value.Type = "array"
					schemaVal.Properties[v.Value.Title] = v
				default:
					return nil, nil, fmt.Errorf("unhandled in container %s: %s", k, v.Value.Type)
				}
			}
			if len(schemaVal.Required) > 0 {
				addAdditionalProperties(schemaVal, AdditionalPropertyUnchanged)
			}

			for k, v := range components.RequestBodies {
				openapiComponents.RequestBodies[k] = v
			}
		} else if dirEntry.IsList() {
			keys := strings.Split(dirEntry.Key, " ")
			listItemPathMultiple := itemPath
			listItemPathSingle := itemPath
			// Add a path for groups of items
			openapiPaths[pathWithPrefix(listItemPathMultiple)] = newPathItem(dirEntry, itemPath, listItemPathMultiple, pathTypeListMultiple, targetAlias)

			for _, k := range keys {
				listItemPathSingle += fmt.Sprintf("/{%s}", k)
			}
			// Add a path for individual items
			openapiPaths[pathWithPrefix(listItemPathSingle)] = newPathItem(dirEntry, itemPath, listItemPathSingle, pathTypeContainer, targetAlias)

			paths, components, err := buildSchema(dirEntry, dirEntry.Config, listItemPathSingle, targetAlias)
			if err != nil {
				return nil, nil, err
			}
			for k, v := range paths {
				openapiPaths[k] = v
			}

			asSingle := openapi3.NewObjectSchema()
			asSingle.Extensions = make(map[string]interface{})
			asSingle.Title = toUnderScore(itemPath)
			asSingle.Description = fmt.Sprintf("%s (single)", dirEntry.Description)
			asSingle.Extensions["x-list-multiple"] = true

			if dirEntry.Extra != nil {
				mustArgs := make([]yang.Must, 0)
				for k, v := range dirEntry.Extra {
					switch k {
					case "must":
						for _, e := range v {
							emap, ok := e.(map[string]interface{})
							if ok {
								m := yang.Must{}
								if name, ok := emap["Name"]; ok {
									m.Name = name.(string)
								} else {
									continue
								}
								if errMsg, ok := emap["ErrorMessage"]; ok {
									if errMsgMap, ok := errMsg.(map[string]interface{}); ok {
										if errMsgName, ok := errMsgMap["Name"]; ok {
											m.ErrorMessage = &yang.Value{
												Name: errMsgName.(string),
											}
										}
									}
								}
								mustArgs = append(mustArgs, m)
							}
						}
					}
				}
				asSingle.Extensions["x-must"] = mustArgs
			}

			openapiComponents.Schemas[toUnderScore(itemPath)] = asSingle.NewRef()

			asMultiple := openapi3.NewArraySchema()
			asMultiple.Items = &openapi3.SchemaRef{
				Ref:   fmt.Sprintf("#/components/schemas/%s", toUnderScore(itemPath)),
				Value: asSingle,
			}
			asMultiple.Extensions = make(map[string]interface{})
			asMultiple.Extensions["x-list-multiple"] = true
			asMultiple.Extensions["x-keys"] = keys
			asMultiple.MinItems = dirEntry.ListAttr.MinElements
			if dirEntry.ListAttr.MaxElements != math.MaxUint64 {
				asMultiple.MaxItems = &dirEntry.ListAttr.MaxElements
			}
			asMultiple.UniqueItems = true
			asMultiple.Description = fmt.Sprintf("%s (list)", dirEntry.Description)

			openapiComponents.Schemas[toUnderscoreWithPathType(itemPath, pathTypeListMultiple)] = asMultiple.NewRef()

			rbRefSingle := &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().WithContent(
					openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{
						Value: asSingle,
						Ref:   fmt.Sprintf("#/components/schemas/%s", toUnderscoreWithPathType(itemPath, pathTypeContainer)),
					}),
				),
			}
			openapiComponents.RequestBodies[fmt.Sprintf("RequestBody_%s", toUnderscoreWithPathType(itemPath, pathTypeContainer))] = rbRefSingle

			if openapiPaths[pathWithPrefix(listItemPathSingle)] != nil &&
				openapiPaths[pathWithPrefix(listItemPathSingle)].Post != nil &&
				openapiPaths[pathWithPrefix(listItemPathSingle)].Post.RequestBody != nil &&
				openapiPaths[pathWithPrefix(listItemPathSingle)].Post.RequestBody.Ref != "" {
				openapiPaths[pathWithPrefix(listItemPathSingle)].Post.RequestBody.Value = rbRefSingle.Value
			}

			respGet200Multiple := openapi3.NewResponse()
			respGet200Multiple.Description = &respGet200Desc
			respGet200Multiple.Content = openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{
				Value: asMultiple,
				Ref:   fmt.Sprintf("#/components/schemas/%s", toUnderscoreWithPathType(itemPath, pathTypeListMultiple)),
			})
			openapiPaths[pathWithPrefix(listItemPathMultiple)].Get.AddResponse(200, respGet200Multiple)

			respGet200 := openapi3.NewResponse()
			respGet200.Description = &respGet200Desc
			respGet200.Content = openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{
				Value: asSingle,
				Ref:   fmt.Sprintf("#/components/schemas/%s", toUnderscoreWithPathType(itemPath, pathTypeContainer)),
			})
			openapiPaths[pathWithPrefix(listItemPathSingle)].Get.AddResponse(200, respGet200)

			if len(strings.Split(itemPath, "/")) <= 2 {
				addAdditionalProperties(asSingle, additionalPropertyTarget(targetAlias))
			}
			for k, v := range components.Schemas {
				switch v.Value.Type {
				case "array": // List as a child of list
					schemaPath := pathToSchemaName(itemPath)
					root := k[len(schemaPath) : len(k)-5] // Remove the _List
					if strings.Count(root, "_") == 0 {
						asSingle.Properties[strings.ToLower(root)] = &openapi3.SchemaRef{
							Ref:   fmt.Sprintf("#/components/schemas/%s", k),
							Value: v.Value,
						}
					}
					openapiComponents.Schemas[k] = v
				case "leaf-list": // Leaf-list as a child of list
					v.Value.Type = "array"
					title := v.Value.Title
					if strings.HasPrefix(title, "leaf-list") {
						title = v.Value.Title[10:]
					}
					asSingle.Properties[title] = openapi3.NewSchemaRef(
						fmt.Sprintf("#/components/schemas/%s", k), v.Value)

					openapiComponents.Schemas[k] = v
				case "object": // Container as a child of list
					if _, ok := v.Value.Extensions["x-list-multiple"]; !ok {
						schemaPath := pathToSchemaName(itemPath)
						root := k[len(schemaPath):]
						if v.Value.Title != "" && !strings.Contains(root, "_") {
							asSingle.Properties[strings.ToLower(lastPartOf(k))] = &openapi3.SchemaRef{
								Ref:   fmt.Sprintf("#/components/schemas/%s", v.Value.Title),
								Value: v.Value,
							}
						}
					}
					openapiComponents.Schemas[k] = v
				case "string", "boolean", "integer", "number": // leaf as a child of list
					if v.Value.Required != nil {
						asSingle.Required = append(asSingle.Required, v.Value.Required...)
						sort.Strings(asSingle.Required)
						v.Value.Required = nil
					}
					asSingle.Properties[v.Value.Title] = v
				default:
					return nil, nil, fmt.Errorf("unhandled in list %s: %s", k, v.Value.Type)
				}
				for _, key := range keys {
					if key == v.Value.Title {
						if v.Value.Extensions == nil {
							v.Value.Extensions = make(map[string]interface{})
						}
						if v.Value.Type == "string" {
							v.Value.Extensions["x-go-type"] = "ListKey"
						}
					}
				}
			}
			if len(asSingle.Required) > 1 {
				addAdditionalProperties(asSingle, AdditionalPropertyUnchanged)
			}
			for k, v := range components.RequestBodies {
				openapiComponents.RequestBodies[k] = v
			}
		} else {
			return nil, nil, fmt.Errorf("unhandled dirEntry %v.Type %v", dirEntry.Name, dirEntry.Type)
		}
	}
	return openapiPaths, &openapiComponents, nil
}

func newPathItem(dirEntry *yang.Entry, itemPath string, parentPath string, pathType pathType, targetAlias string) *openapi3.PathItem {
	getOp := openapi3.NewOperation()
	getOp.Summary = fmt.Sprintf("GET %s %s", itemPath, pathType.string())
	getOp.OperationID = fmt.Sprintf("get%s_%s", toUnderScore(itemPath), toUnderScore(pathType.string()))
	if pathType == pathTypeContainer {
		getOp.OperationID = fmt.Sprintf("get%s", toUnderScore(itemPath))
	}
	getOp.Responses = make(openapi3.Responses)

	parameters := make(openapi3.Parameters, 0)
	pathKeys := strings.Split(parentPath, "/")
	targetParameterRef := openapi3.ParameterRef{
		Ref:   fmt.Sprintf("#/components/parameters/%s", targetAlias),
		Value: targetParameter.Value,
	}
	parameters = append(parameters, &targetParameterRef)

	parameterNames := make(map[string]interface{})

	for _, pathKey := range pathKeys {
		if strings.HasPrefix(pathKey, "{") && strings.HasSuffix(pathKey, "}") {
			k := pathKey[1 : len(pathKey)-1]
			pathKey := pathKey // pinning
			if _, alreadyUsed := parameterNames[k]; alreadyUsed {
				newK := fmt.Sprintf("%s_%d", k, len(parameterNames))
				pathKey = fmt.Sprintf("{%s}", newK)
				// TODO: add the changed parameter name back in to the stored path
				k = newK
			}
			p := openapi3.ParameterRef{
				//Ref: k,
				Value: openapi3.NewPathParameter(k),
			}
			p.Value.Description = fmt.Sprintf("key %s", pathKey)
			p.Value.Content = openapi3.NewContent()
			mt := openapi3.NewMediaType()
			mt.Schema = &openapi3.SchemaRef{
				Value: openapi3.NewStringSchema(),
			}
			p.Value.Content["text/plain; charset=utf-8"] = mt
			parameterNames[k] = struct{}{}
			parameters = append(parameters, &p)
		}
	}

	newPath := openapi3.PathItem{
		Get:         getOp,
		Description: dirEntry.Description,
		Parameters:  parameters,
	}
	if dirEntry.Kind == yang.ChoiceEntry {
		newPath.Description = fmt.Sprintf("YANG Choice: %s", dirEntry.Name)
	} else if dirEntry.Kind == yang.CaseEntry {
		newPath.Description = fmt.Sprintf("YANG Choice Case: %s", dirEntry.Name)
	}

	if dirEntry.Config != yang.TSFalse && dirEntry.Parent.Config != yang.TSFalse && pathType != pathTypeListMultiple {
		deleteOp := openapi3.NewOperation()
		deleteOp.Summary = fmt.Sprintf("DELETE %s", itemPath)
		deleteOp.OperationID = fmt.Sprintf("delete%s_%s", toUnderScore(itemPath), toUnderScore(pathType.string()))
		if pathType == pathTypeContainer {
			deleteOp.OperationID = fmt.Sprintf("delete%s", toUnderScore(itemPath))
		}
		del20Ok := "DELETE 200 OK"
		deleteResp200 := &openapi3.Response{
			Description: &del20Ok,
		}
		deleteOp.Responses = openapi3.Responses{"200": &openapi3.ResponseRef{
			Value: deleteResp200,
		}}
		newPath.Delete = deleteOp

		postOp := openapi3.NewOperation()
		postOp.Summary = fmt.Sprintf("POST %s", itemPath)
		postOp.OperationID = fmt.Sprintf("post%s", toUnderScore(itemPath))
		postOp.Responses = make(openapi3.Responses)
		postOp.Responses["201"] = &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("created")}
		postOp.RequestBody = &openapi3.RequestBodyRef{
			Ref: fmt.Sprintf("#/components/requestBodies/RequestBody_%s", toUnderScore(itemPath)),
			// Value is filled in later
		}
		newPath.Post = postOp
	}

	return &newPath
}

func toUnderScore(itemPath string) string {
	pathParts := make([]string, 0)
	for _, pathPart := range strings.Split(itemPath, "/") {
		if pathPart == "" || strings.HasPrefix(pathPart, "{") || strings.HasSuffix(pathPart, "}") {
			continue
		}
		pathParts = append(pathParts, uppercaseFirstCharacter(pathPart))
	}

	return strings.Join(pathParts, "_")
}

func lastPartOf(path string) string {
	pathParts := strings.Split(path, "_")
	return pathParts[len(pathParts)-1]
}

// Uppercase the first character in a string. This assumes UTF-8, so we have
// to be careful with unicode, don't treat it as a byte array.
func uppercaseFirstCharacter(str string) string {
	if str == "" {
		return ""
	}
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func pathWithPrefix(itemPath string) string {
	return fmt.Sprintf("%s%s", pathPrefix, itemPath)
}

// Removes any indices
func pathToSchemaName(itemPath string) string {
	parts := strings.Split(itemPath, "/")
	partsWoIdx := make([]string, 0)
	for _, p := range parts {
		if !strings.Contains(p, "{") {
			partsWoIdx = append(partsWoIdx, p)
		}
	}
	return strings.ToLower(strings.Join(partsWoIdx, "/"))
}

func toUnderscoreWithPathType(itemPath string, pathType pathType) string {
	if pathType == pathTypeListMultiple {
		return fmt.Sprintf("%s_%s", toUnderScore(itemPath), uppercaseFirstCharacter(pathType.string()))
	}
	return toUnderScore(itemPath)
}

// If there is more than 1 range - try to find the overall min and max
// If YANG uses min and max, then leave out any statement in OpenAPI 3
// Leave it to the implementation handle the min and max for the type
func yangRange(yangRange yang.YangRange, parentType yang.TypeKind) (*float64, *float64, error) {
	var minVal = math.MaxFloat64
	var maxVal = -math.MaxFloat64
	var hasMinMin, hasMaxMax bool
	if yangRange.Len() == 0 {
		return nil, nil, fmt.Errorf("unexpected nil range")
	}
	for i := 0; i < yangRange.Len(); i++ {
		newMinVal := floatFromYnumber(yangRange[i].Min)
		if newMinVal < minVal {
			minVal = newMinVal
		}
		newMaxVal := floatFromYnumber(yangRange[i].Max)
		if newMaxVal > maxVal {
			maxVal = newMaxVal
		}
		switch parentType {
		case yang.Yint32:
			if floatFromYnumber(yangRange[i].Min) == math.MinInt32 {
				hasMinMin = true
			}
			if floatFromYnumber(yangRange[i].Max) == math.MaxInt32 {
				hasMaxMax = true
			}
		case yang.Yint64:
			if floatFromYnumber(yangRange[i].Min) == math.MinInt64 {
				hasMinMin = true
			}
			if floatFromYnumber(yangRange[i].Max) == math.MaxInt64 {
				hasMaxMax = true
			}
		case yang.Yuint32:
			if floatFromYnumber(yangRange[i].Max) == math.MaxUint32 {
				hasMaxMax = true // openapi will limit value to int32
			}
		case yang.Yuint64:
			if floatFromYnumber(yangRange[i].Max) == math.MaxUint64 {
				hasMaxMax = true
			}
		case yang.Ystring:
			if floatFromYnumber(yangRange[i].Max) == 18446744073709551615 {
				hasMaxMax = true
			}
		}
	}
	if hasMinMin && hasMaxMax {
		return nil, nil, nil
	} else if hasMinMin && !hasMaxMax {
		return nil, &maxVal, nil
	} else if !hasMinMin && hasMaxMax {
		return &minVal, nil, nil
	}
	return &minVal, &maxVal, nil
}

func yangDefault(leaf *yang.Entry) (interface{}, error) {
	if leaf.Type.Default != "" {
		switch leaf.Type.Kind {
		case yang.Yint8:
			intValue, err := strconv.ParseInt(leaf.Type.Default, 10, 8)
			return int8(intValue), err
		case yang.Yuint8:
			intValue, err := strconv.ParseUint(leaf.Type.Default, 10, 8)
			return uint8(intValue), err
		case yang.Yint16:
			intValue, err := strconv.ParseInt(leaf.Type.Default, 10, 16)
			return int16(intValue), err
		case yang.Yuint16:
			intValue, err := strconv.ParseUint(leaf.Type.Default, 10, 16)
			return uint16(intValue), err
		case yang.Yint32:
			intValue, err := strconv.ParseInt(leaf.Type.Default, 10, 32)
			return int32(intValue), err
		case yang.Yuint32:
			intValue, err := strconv.ParseUint(leaf.Type.Default, 10, 32)
			return uint32(intValue), err
		case yang.Yint64:
			intValue, err := strconv.ParseInt(leaf.Type.Default, 10, 64)
			return int64(intValue), err
		case yang.Yuint64:
			intValue, err := strconv.ParseUint(leaf.Type.Default, 10, 64)
			return uint64(intValue), err
		case yang.Ydecimal64:
			return strconv.ParseFloat(leaf.Type.Default, 64)
		}
	}
	return nil, nil
}

func floatFromYnumber(ynumber yang.Number) float64 {
	neg := 1.0
	if ynumber.Negative {
		neg = -1.0
	}
	v := float64(ynumber.Value) * neg * math.Pow(10, -1.0*float64(ynumber.FractionDigits))
	return v
}

func additionalPropertyTarget(targetAlias string) string {
	caser := cases.Title(language.English)
	return fmt.Sprintf("AdditionalProperty%s", caser.String(targetAlias))
}

func resolveLeafRefType(leaf *yang.Entry) yang.TypeKind {
	path := leaf.Type.Path
	if strings.HasPrefix(path, "..") {
		return walkPath(leaf.Parent, path[3:])
	} else if strings.HasPrefix(path, "//") {
		fmt.Println("unhandled relative path")
	} else if strings.HasPrefix(path, "/") {
		root := resolveLeafRefRoot(leaf)
		return walkPath(root, path[1:])
	}
	return yang.Ystring
}

func resolveLeafRefRoot(entry *yang.Entry) *yang.Entry {
	if entry.Parent != nil {
		return resolveLeafRefRoot(entry.Parent)
	}
	return entry
}

func walkPath(entry *yang.Entry, path string) yang.TypeKind {
	if path == "" {
		return entry.Type.Kind
	} else if strings.HasPrefix(path, "../") {
		return walkPath(entry.Parent, path[3:])
	}

	pathParts := strings.Split(path, "/")
	colonIdx := strings.Index(pathParts[0], ":")
	var prefix, name string
	if colonIdx == -1 {
		name = pathParts[0]
	} else {
		prefix = pathParts[0][:colonIdx]
		name = pathParts[0][colonIdx+1:]
	}
	childEntry, ok := entry.Dir[name]
	if !ok {
		return yang.Ystring
	}
	if childEntry.Prefix.Name != prefix && colonIdx > -1 {
		return yang.Ystring
	}
	return walkPath(childEntry, strings.Join(pathParts[1:], "/"))
}
