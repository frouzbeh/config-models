// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package navigator

import (
	"github.com/SeanCondon/xpath"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type testDevice struct {
	TestStruct *testDevice_testStruct `path:"testStruct"`
}

func (td *testDevice) IsYANGGoStruct() {
}

func (td *testDevice) Validate(...ygot.ValidationOption) error {
	return nil
}

func (td *testDevice) ΛEnumTypeMap() map[string][]reflect.Type {
	return nil
}

func (td *testDevice) ΛBelongingModule() string {
	return ""
}

type testDevice_testStruct struct {
	A interface{} `path:"a"`
	B interface{} `path:"b"`
	C interface{} `path:"c"`
	D interface{} `path:"d"`
}

func Test_Value(t *testing.T) {
	aValue := "test1"
	bValue := 10
	dValue := true
	testStruct1 := testDevice_testStruct{
		A: &aValue,
		B: &bValue,
		D: &dValue,
	}

	td := testDevice{
		TestStruct: &testStruct1,
	}

	entry := &yang.Entry{
		Name: "testDevice",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"testStruct": {
				Name: "testStruct",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"a": {
						Name: "a",
						Kind: yang.LeafEntry,
					},
					"b": {
						Name: "b",
						Kind: yang.LeafEntry,
					},
					"c": { // C is present in the metadata but nil in struct
						Name: "c",
						Kind: yang.LeafEntry,
					},
					"d": {
						Name: "d",
						Kind: yang.LeafEntry,
					},
				},
			},
		},
	}

	nn := NewYangNodeNavigator(entry, &td, false)
	assert.NotNil(t, nn)
	assert.Equal(t, "testDevice", nn.LocalName())
	assert.Equal(t, "value of testDevice", nn.Value())
	assert.Equal(t, xpath.ElementNode, nn.NodeType())

	movedChild1 := nn.MoveToChild()
	assert.True(t, movedChild1)

	assert.Equal(t, "testStruct", nn.LocalName())
	assert.Equal(t, "value of testStruct", nn.Value())
	assert.Equal(t, xpath.ElementNode, nn.NodeType())

	movedChild2 := nn.MoveToChild()
	assert.True(t, movedChild2)
	assert.Equal(t, "a", nn.LocalName())
	assert.Equal(t, "test1", nn.Value())
	assert.Equal(t, xpath.ElementNode, nn.NodeType())

	movedChild3 := nn.MoveToChild()
	assert.False(t, movedChild3)

	moveNext1 := nn.MoveToNext()
	assert.True(t, moveNext1)
	assert.Equal(t, "b", nn.LocalName())
	assert.Equal(t, "10", nn.Value())
	assert.Equal(t, xpath.ElementNode, nn.NodeType())

	moveNext2 := nn.MoveToNext() // Goes to D because C is nil in struct
	assert.True(t, moveNext2)
	assert.Equal(t, "d", nn.LocalName())
	assert.Equal(t, "true", nn.Value())
	assert.Equal(t, xpath.ElementNode, nn.NodeType())

	movePrevious1 := nn.MoveToPrevious() // Goes back to B
	assert.True(t, movePrevious1)
	assert.Equal(t, "b", nn.LocalName())
	assert.Equal(t, "10", nn.Value())

	moveFirst1 := nn.MoveToFirst() // Goes back to A
	assert.True(t, moveFirst1)
	assert.Equal(t, "a", nn.LocalName())
	assert.Equal(t, "test1", nn.Value())

	nn.MoveToRoot()

}

func Test_processStruct(t *testing.T) {
	aValue := "test1"
	bValue := 10
	testStruct1 := &testDevice_testStruct{
		A: &aValue,
		B: &bValue,
	}

	entry := &yang.Entry{
		Name: "a",
		Kind: yang.LeafEntry,
	}

	testStructValue := reflect.ValueOf(testStruct1)

	// Processing the struct should add the value on to the yang.Entry
	// as 'gostruct' in Annotation
	processedEntry := processStruct(testStructValue, "a", entry)
	assert.NotNil(t, processedEntry)
	assert.Equal(t, 1, len(processedEntry))
	processedEntryA, ok := processedEntry["a"]
	assert.True(t, ok)
	assert.Equal(t, yang.LeafEntry, processedEntryA.Kind)
	assert.Equal(t, "a", processedEntryA.Name)
	// Get the annotations
	assert.Equal(t, 1, len(processedEntryA.Annotation))
	value, valueOk := processedEntryA.Annotation[goStruct]
	assert.True(t, valueOk)
	valStr, typeOK := value.(*string)
	assert.True(t, typeOK)
	assert.Equal(t, "test1", *valStr)
}

func Test_getOrderedKeys(t *testing.T) {
	annotation1 := make(map[string]interface{})
	attribs := []string{"a", "b", "c", "d"}
	annotation1[orderedAttrList] = attribs

	orderedKeys := getOrderedKeys(annotation1)
	assert.Equal(t, 4, len(orderedKeys))
	assert.EqualValues(t, attribs, orderedKeys)
}

func Test_getNextKey(t *testing.T) {
	key := "testKey"
	selfAnnotation := make(map[string]interface{})
	selfAnnotation[key] = "b"

	parentAnnotation := make(map[string]interface{})
	attribs := []string{"testKey__a", "testKey__b", "testKey__c", "testKey__d"}
	parentAnnotation[orderedAttrList] = attribs

	assert.Equal(t, "testKey__c", getNextKey(selfAnnotation, parentAnnotation, key))
}

func Test_getPreviousKey(t *testing.T) {
	key := "b"

	annotation := make(map[string]interface{})
	attribs := []string{"a", "b", "c", "d"}
	annotation[orderedAttrList] = attribs

	assert.Equal(t, "a", getPreviousKey(annotation, key))
}

func Test_getGoStruct(t *testing.T) {
	someValue := 32
	annotation := make(map[string]interface{})
	annotation[goStruct] = &someValue

	assert.Equal(t, &someValue, getGoStruct(annotation))
}

func Test_extractMust(t *testing.T) {
	mustAsExtra := map[string]interface{}{
		"Name": "1 = 1",
		"Description": map[string]interface{}{
			"Name": "sample description",
		},
		"ErrorMessage": map[string]interface{}{
			"Name": "sample error message",
		},
		"ErrorAppTag": map[string]interface{}{
			"Name": "sample error app tag",
		},
	}

	extras := []interface{}{
		mustAsExtra,
	}

	mustStmt := extractMust(extras)
	assert.NotNil(t, mustStmt)
	assert.Equal(t, "1 = 1", mustStmt.Name)
	assert.Equal(t, "sample description", mustStmt.Description.Name)
	assert.Equal(t, "sample error message", mustStmt.ErrorMessage.Name)
	assert.Equal(t, "sample error app tag", mustStmt.ErrorAppTag.Name)
}

func Test_deepCopyDir(t *testing.T) {
	parentDir := &yang.Entry{
		Name: "sample-parent",
		Dir:  make(map[string]*yang.Entry),
	}

	sampleDir := &yang.Entry{
		Parent:      parentDir,
		Node:        nil,
		Name:        "sample-dir",
		Description: "this is a sample yang entry",
		Default:     []string{"2"},
		Units:       "mm",
		Errors:      nil,
		Kind:        yang.LeafEntry,
		Config:      0,
		Prefix: &yang.Value{
			Name: "t1",
		},
		Mandatory: yang.TSTrue,
		Dir: map[string]*yang.Entry{
			// We would not expect to have child Dirs when it's a Leaf, but this is just for testing
			"someValue": {
				Name:        "someValue",
				Description: "description of child dir",
			},
			"someother__value": { // This will not be copied, as it is a list entry __
				Name: "someother__value",
			},
		},
		Key: "",
		Type: &yang.YangType{
			Name: "number",
		},
		Exts:       nil,
		ListAttr:   nil,
		RPC:        nil,
		Identities: nil,
		Augments:   nil,
		Augmented:  nil,
		Deviations: nil,
		Deviate:    nil,
		Uses:       nil,
		Extra:      nil,
		Annotation: map[string]interface{}{
			"otherValue":    10,
			goStruct:        "test value",       // should not be copied
			orderedAttrList: []string{"a", "b"}, // should not be copied
		},
	}
	parentDir.Dir["sample-dir"] = sampleDir

	copiedDir := deepCopyDir(sampleDir)

	assert.NotNil(t, copiedDir)
	assert.Equal(t, "sample-parent", copiedDir.Parent.Name)
	assert.Equal(t, "sample-dir", copiedDir.Name)
	assert.Equal(t, "this is a sample yang entry", copiedDir.Description)
	assert.Equal(t, []string{"2"}, copiedDir.Default)
	assert.Equal(t, "mm", copiedDir.Units)
	assert.Equal(t, yang.LeafEntry, copiedDir.Kind)
	assert.Equal(t, "t1", copiedDir.Prefix.Name)
	assert.Equal(t, "number", copiedDir.Type.Name)

	assert.Equal(t, 1, len(copiedDir.Annotation))
	assert.Equal(t, 10, copiedDir.Annotation["otherValue"])

	assert.Equal(t, 1, len(copiedDir.Dir))
	assert.Equal(t, "someValue", copiedDir.Dir["someValue"].Name)
	assert.Equal(t, "description of child dir", copiedDir.Dir["someValue"].Description)
}

func Test_generateMustError(t *testing.T) {
	aValue := "test1"
	bValue := 10
	testStruct1 := testDevice_testStruct{
		A: &aValue,
		B: &bValue,
	}

	td := testDevice{
		TestStruct: &testStruct1,
	}

	entry := &yang.Entry{
		Name: "testDevice",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"testStruct": {
				Name: "testStruct",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"a": {
						Name: "a",
						Kind: yang.LeafEntry,
					},
					"b": {
						Name: "b",
						Kind: yang.LeafEntry,
					},
				},
			},
		},
	}

	nn := NewYangNodeNavigator(entry, &td, false)
	assert.NotNil(t, nn)

	ynn, ok := nn.(*YangNodeNavigator)
	assert.True(t, ok)
	movedChild1 := ynn.MoveToChild()
	assert.True(t, movedChild1)

	parts := ynn.generateMustError("*")
	assert.Equal(t, 2, len(parts))
	assert.Equal(t, "a=test1", parts[0])
	assert.Equal(t, "b=10", parts[1])
}
