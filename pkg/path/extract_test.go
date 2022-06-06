/*
 * SPDX-FileCopyrightText: 2022-present Intel Corporation
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package path

import (
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	var err error
	schemaTree, err := ygot.GzipToSchema(testdevice10XSchema)
	if err != nil {
		panic(err)
	}

	roPaths, rwPaths = ExtractPaths(schemaTree)
	if err != nil {
		panic(err)
	}

	exitVal := m.Run()

	os.Exit(exitVal)
}

func Test_ExtractPaths(t *testing.T) {
	assert.Equal(t, 2, len(roPaths))
	for _, roPath := range roPaths {
		switch path := roPath.Path; path {
		case "/cont1a/cont2a/leaf2c":
			assert.Equal(t, 1, len(roPath.SubPath))
			sp := roPath.SubPath[0]
			assert.Equal(t, "leaf2c", sp.AttrName)
		case "/cont1b-state":
			assert.Equal(t, 3, len(roPath.SubPath))
			for _, sp := range roPath.SubPath {
				switch subPath := sp.SubPath; subPath {
				case "/leaf2d":
					assert.Equal(t, "leaf2d", sp.AttrName)
				case "/list2b[index=*]/index":
					assert.Equal(t, "index", sp.AttrName)
				case "/list2b[index=*]/leaf3c":
					assert.Equal(t, "leaf3c", sp.AttrName)
				default:
					t.Fatalf("unexpected subpath %s for RO path %s", subPath, path)
				}
			}
		default:
			t.Fatalf("unexpected RO path %s", path)
		}
	}

	assert.Equal(t, 21, len(rwPaths))
	for _, rwPath := range rwPaths {
		switch path := rwPath.Path; path {
		case "/leafAtTopLevel":
			assert.Equal(t, "leafAtTopLevel", rwPath.AttrName)
		case "/cont1a/cont2a/leaf2a":
			assert.Equal(t, "leaf2a", rwPath.AttrName)
		case "/cont1a/cont2a/leaf2b":
			assert.Equal(t, "leaf2b", rwPath.AttrName)
		case "/cont1a/cont2a/leaf2d":
			assert.Equal(t, "leaf2d", rwPath.AttrName)
		case "/cont1a/cont2a/leaf2e":
			assert.Equal(t, "leaf2e", rwPath.AttrName)
		case "/cont1a/cont2a/leaf2f":
			assert.Equal(t, "leaf2f", rwPath.AttrName)
		case "/cont1a/cont2a/leaf2g":
			assert.Equal(t, "leaf2g", rwPath.AttrName)
		case "/cont1a/leaf1a":
			assert.Equal(t, "leaf1a", rwPath.AttrName)
		case "/cont1a/list4[id=*]/id":
			assert.Equal(t, "id", rwPath.AttrName)
		case "/cont1a/list4[id=*]/leaf4b":
			assert.Equal(t, "leaf4b", rwPath.AttrName)
		case "/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1":
			assert.Equal(t, "fkey1", rwPath.AttrName)
		case "/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey2":
			assert.Equal(t, "fkey2", rwPath.AttrName)
		case "/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/displayname":
			assert.Equal(t, "displayname", rwPath.AttrName)
		case "/cont1a/list2a[name=*]/name":
			assert.Equal(t, "name", rwPath.AttrName)
		case "/cont1a/list2a[name=*]/tx-power":
			assert.Equal(t, "tx-power", rwPath.AttrName)
		case "/cont1a/list2a[name=*]/ref2d":
			assert.Equal(t, "ref2d", rwPath.AttrName)
		case "/cont1a/list2a[name=*]/range-min":
			assert.Equal(t, "range-min", rwPath.AttrName)
		case "/cont1a/list2a[name=*]/range-max":
			assert.Equal(t, "range-max", rwPath.AttrName)
		case "/cont1a/list5[key1=*][key2=*]/key1":
			assert.Equal(t, "key1", rwPath.AttrName)
		case "/cont1a/list5[key1=*][key2=*]/key2":
			assert.Equal(t, "key2", rwPath.AttrName)
		case "/cont1a/list5[key1=*][key2=*]/leaf5a":
			assert.Equal(t, "leaf5a", rwPath.AttrName)
		default:
			t.Fatalf("unexpected RW path %s", path)
		}
	}
}

func Test_formatNameAsPath(t *testing.T) {
	type formatNameTest struct {
		testName      string
		name          string
		key           string
		parent        string
		subpathPrefix string
		expected      string
	}
	tests := []formatNameTest{
		{
			testName:      "3 indices",
			name:          "test-entry-1",
			key:           "test-index1 test-index2 test-index3",
			parent:        "test-parent",
			subpathPrefix: "/spp",
			expected:      "test-parent/spp/test-entry-1[test-index1=*][test-index2=*][test-index3=*]",
		},
		{
			testName: "simple",
			name:     "test-entry-2",
			parent:   "test-parent",
			expected: "test-parent/test-entry-2",
		}, {
			testName: "basic",
			name:     "foo",
			expected: "/foo",
		},
		{
			testName:      "parent-and-subpath",
			name:          "foo",
			parent:        "/parentPath",
			subpathPrefix: "/subpathPrefix",
			expected:      "/parentPath/subpathPrefix/foo",
		},
		{
			testName: "list",
			name:     "foo",
			key:      "key1 key2",
			expected: "/foo[key1=*][key2=*]",
		},
	}

	for _, tt := range tests {
		dirEntry := &yang.Entry{
			Name: tt.name,
			Key:  tt.key,
		}
		if tt.key != "" {
			dirEntry.ListAttr = new(yang.ListAttr)
			dirEntry.Dir = make(map[string]*yang.Entry)
		}
		formatted := formatNameAsPath(dirEntry, tt.parent, tt.subpathPrefix)
		assert.Equal(t, tt.expected, formatted, tt.testName)
	}
}

func Test_earliestRoAncestor(t *testing.T) {
	type earliestAncestorTest struct {
		name            string
		dirEntry        *yang.Entry
		expectedBase    []string
		expectedSubpath []string
		expectedFalse   bool
	}
	tests := []earliestAncestorTest{
		{
			name: "level 3 of 4",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSUnset, // This stays at config=false because of it parent (below)
				Parent: &yang.Entry{
					Name:   "parent",
					Config: yang.TSUnset, // This stays at config=false because of it parent (below)
					Parent: &yang.Entry{
						Name:   "grandparent",
						Config: yang.TSFalse, // This is the one that changes it to config=false
						Parent: &yang.Entry{
							Name:   "great-grandparent",
							Config: yang.TSUnset, // This is a config node by default
						},
					},
				},
			},
			expectedBase:    []string{"great-grandparent", "grandparent"},
			expectedSubpath: []string{"parent", "child"},
			expectedFalse:   true,
		},
		{
			name: "level 3 of 4 - double marked",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSFalse, // This is marked specifically as config=false. unnecessary because of its parent (below), but can happen
				Parent: &yang.Entry{
					Name:   "parent",
					Config: yang.TSFalse, // This is marked specifically as config=false. unnecessary because of its parent (below), but can happen
					Parent: &yang.Entry{
						Name:   "grandparent",
						Config: yang.TSFalse, // This is the one that changes it to config=false
						Parent: &yang.Entry{
							Name:   "great-grandparent",
							Config: yang.TSUnset, // This is a config node by default
						},
					},
				},
			},
			expectedBase:    []string{"great-grandparent", "grandparent"},
			expectedSubpath: []string{"parent", "child"},
			expectedFalse:   true,
		},
		{
			name: "level 2 of 4",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSUnset, // This stays at config=false because of it parent (below)
				Parent: &yang.Entry{
					Name:   "parent",
					Config: yang.TSFalse, // This is the one that changes it to config=false
					Parent: &yang.Entry{
						Name:   "grandparent",
						Config: yang.TSUnset, // This is a config=true node by default
						Parent: &yang.Entry{
							Name:   "great-grandparent",
							Config: yang.TSUnset, // This is a config=true node by default
						},
					},
				},
			},
			expectedBase:    []string{"great-grandparent", "grandparent", "parent"},
			expectedSubpath: []string{"child"},
			expectedFalse:   true,
		},
		{
			name: "level 1 of 4",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSFalse, // This stays at config=false because of it parent (below)
				Parent: &yang.Entry{
					Name:   "parent",
					Config: yang.TSUnset, // This is the one that changes it to config=false
					Parent: &yang.Entry{
						Name:   "grandparent",
						Config: yang.TSUnset, // This is a config=true node by default
						Parent: &yang.Entry{
							Name:   "great-grandparent",
							Config: yang.TSUnset, // This is a config node by default
						},
					},
				},
			},
			expectedBase:    []string{"great-grandparent", "grandparent", "parent", "child"},
			expectedSubpath: nil,
			expectedFalse:   true,
		},
		{
			name: "with list index",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSFalse, // This stays at config=false because of it parent (below)
				Parent: &yang.Entry{
					Name:   "parent-list",
					Config: yang.TSUnset, // This is the one that changes it to config=false
					ListAttr: &yang.ListAttr{
						MinElements: 0,
						MaxElements: 0,
						OrderedBy:   nil,
					},
					Key: "id",
					Dir: make(map[string]*yang.Entry),
					Parent: &yang.Entry{
						Name:   "grandparent",
						Config: yang.TSUnset, // This is a config=true node by default
						Parent: &yang.Entry{
							Name:   "great-grandparent",
							Config: yang.TSUnset, // This is a config node by default
						},
					},
				},
			},
			expectedBase:    []string{"great-grandparent", "grandparent", "parent-list[id=*]", "child"},
			expectedSubpath: nil,
			expectedFalse:   true,
		},
		{
			name: "no readonly",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSUnset, // This is a config=true node by default
				Parent: &yang.Entry{
					Name:   "parent",
					Config: yang.TSUnset, // This is a config=true node by default
					Parent: &yang.Entry{
						Name:   "grandparent",
						Config: yang.TSUnset, // This is a config=true node by default
						Parent: &yang.Entry{
							Name:   "great-grandparent",
							Config: yang.TSUnset, // This is a config=true node by default
						},
					},
				},
			},
			expectedBase:    []string{"great-grandparent", "grandparent", "parent", "child"},
			expectedSubpath: nil,
			expectedFalse:   false,
		},
		{
			name: "top level",
			dirEntry: &yang.Entry{
				Name:   "child",
				Config: yang.TSFalse, // This is the one that changes it to config=false
			},
			expectedBase:    []string{"child"},
			expectedSubpath: nil,
			expectedFalse:   true,
		},
	}

	for _, tt := range tests {
		base, subpath, configFalse := earliestRoAncestor(tt.dirEntry)
		assert.Equal(t, tt.expectedBase, base, tt.name)
		assert.Equal(t, tt.expectedSubpath, subpath, tt.name)
		assert.Equal(t, tt.expectedFalse, configFalse, tt.name)
	}
}

func Test_formatNameOfChildEntry(t *testing.T) {
	type testFormat struct {
		testName string
		dirEntry *yang.Entry
		expected string
	}
	tests := []testFormat{
		{
			testName: "simple",
			dirEntry: &yang.Entry{
				Name: "test",
			},
			expected: "test",
		},
		{
			testName: "with double key",
			dirEntry: &yang.Entry{
				Name:     "test",
				ListAttr: new(yang.ListAttr),
				Dir:      make(map[string]*yang.Entry),
				Key:      "idx1 idx2",
			},
			expected: "test[idx1=*][idx2=*]",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatNameOfChildEntry(tt.dirEntry), tt.testName)
	}
}

var (
	// testdevice10XSchema is a byte slice contain a gzip compressed representation of the
	// YANG schema from which the Go code was generated. When uncompressed the
	// contents of the byte slice is a JSON document containing an object, keyed
	// on the name of the generated struct, and containing the JSON marshalled
	// contents of a goyang yang.Entry struct, which defines the schema for the
	// fields within the struct.
	testdevice10XSchema = []byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xec, 0x5d, 0x5b, 0x6f, 0xdb, 0x38,
		0x16, 0x7e, 0xcf, 0xaf, 0x38, 0xd0, 0x4b, 0x93, 0x45, 0x9c, 0xd8, 0x8a, 0x9d, 0xb6, 0x01, 0x06,
		0x58, 0xf7, 0xb6, 0x0b, 0x4c, 0xdb, 0x29, 0x3a, 0xc1, 0x00, 0xbb, 0xdd, 0x60, 0x41, 0x5b, 0x47,
		0x36, 0x11, 0x99, 0xf2, 0x48, 0x54, 0x1a, 0x63, 0x90, 0xff, 0x3e, 0xd0, 0xc5, 0xf2, 0x4d, 0x96,
		0x0e, 0x75, 0x71, 0xec, 0x98, 0xf3, 0x30, 0x75, 0x64, 0x4a, 0x16, 0x79, 0xae, 0xfc, 0xce, 0x47,
		0xf2, 0xaf, 0x13, 0x00, 0x00, 0xe3, 0x2b, 0x9b, 0xa0, 0x71, 0x03, 0x86, 0x85, 0x0f, 0x7c, 0x88,
		0xc6, 0x79, 0x7c, 0xf5, 0x57, 0x2e, 0x2c, 0xe3, 0x06, 0x3a, 0xc9, 0x9f, 0xef, 0x5d, 0x61, 0xf3,
		0x91, 0x71, 0x03, 0xed, 0xe4, 0xc2, 0x07, 0xee, 0x19, 0x37, 0x10, 0x3f, 0x22, 0xba, 0x30, 0x74,
		0x85, 0xec, 0xb0, 0x95, 0x6b, 0x2b, 0x8f, 0x4f, 0xbe, 0x3f, 0x5f, 0xfd, 0xf6, 0x03, 0xfa, 0x43,
		0x8f, 0x4f, 0x25, 0x77, 0x45, 0xd8, 0xe8, 0x76, 0x8c, 0x20, 0xdd, 0x29, 0x38, 0xf8, 0x80, 0x0e,
		0x84, 0xb7, 0x30, 0x2e, 0xd0, 0x5b, 0xbf, 0x6b, 0xf5, 0xe5, 0xd2, 0xcb, 0xeb, 0x2f, 0x99, 0x7e,
		0xf1, 0xcd, 0x43, 0x9b, 0x3f, 0x6e, 0xbc, 0xdb, 0xca, 0xfb, 0xc9, 0xce, 0xda, 0xaf, 0x44, 0xdf,
		0xfe, 0xee, 0x06, 0xde, 0x10, 0x33, 0xef, 0x8c, 0xdf, 0x04, 0x67, 0x3f, 0x5d, 0x2f, 0x7c, 0x19,
		0x63, 0x1a, 0xff, 0xc8, 0x79, 0x76, 0xc3, 0x7f, 0x33, 0xbf, 0xef, 0x8d, 0x82, 0x09, 0x0a, 0x69,
		0xdc, 0x80, 0xf4, 0x02, 0xdc, 0xd2, 0x70, 0xa9, 0x55, 0xf8, 0x4e, 0x1b, 0x8d, 0x9e, 0x56, 0xae,
		0x3c, 0xad, 0x8f, 0xe7, 0x9a, 0x58, 0x56, 0xc4, 0x63, 0xb2, 0xed, 0x1d, 0x59, 0x16, 0x93, 0xc9,
		0xb6, 0xf5, 0x22, 0x43, 0x5c, 0xa6, 0xb0, 0x0a, 0xc4, 0x55, 0x20, 0xb6, 0x42, 0xf1, 0x51, 0xc4,
		0x48, 0x13, 0x27, 0x55, 0xac, 0xca, 0xe2, 0x55, 0x16, 0x33, 0x59, 0xdc, 0xd9, 0x62, 0xdf, 0x22,
		0xfe, 0x42, 0x35, 0x48, 0x1b, 0x38, 0xc8, 0xec, 0x1c, 0x75, 0xd8, 0x18, 0xce, 0xa4, 0x7d, 0x41,
		0x67, 0xd6, 0xd4, 0xe3, 0x6b, 0x30, 0x41, 0x8f, 0x0f, 0x21, 0xbc, 0x19, 0xb8, 0xf0, 0xb9, 0x85,
		0xf0, 0x7e, 0xae, 0x24, 0x40, 0x79, 0x9c, 0xcd, 0x02, 0x27, 0x1c, 0x9a, 0x1f, 0xb9, 0x0d, 0xa3,
		0xc6, 0xa6, 0x91, 0xdb, 0xe6, 0xae, 0xe0, 0xb7, 0x12, 0xdd, 0x6c, 0x17, 0x34, 0x2b, 0xd2, 0x51,
		0x15, 0x5d, 0x55, 0xd3, 0x59, 0x55, 0xdd, 0x2d, 0xad, 0xc3, 0xa5, 0x75, 0x59, 0x59, 0xa7, 0xf3,
		0x75, 0xbb, 0x40, 0xc7, 0xd3, 0x5f, 0xbb, 0x9d, 0x4d, 0x51, 0x6d, 0x9c, 0x03, 0x2e, 0xe4, 0x1b,
		0xca, 0x50, 0x27, 0x4a, 0xd1, 0x23, 0x34, 0xfd, 0xce, 0xc4, 0x08, 0x49, 0x9a, 0x1a, 0xfe, 0x47,
		0x13, 0x5d, 0xf4, 0xe0, 0x2f, 0x5c, 0x90, 0x65, 0x9d, 0xde, 0xf4, 0x07, 0x73, 0x02, 0xdc, 0xee,
		0x6a, 0xb7, 0xde, 0xf7, 0xc9, 0x63, 0xc3, 0xd0, 0x7a, 0x3f, 0xf0, 0x11, 0x97, 0x7e, 0xb1, 0x9a,
		0x6f, 0x0e, 0x31, 0x8e, 0x98, 0xe4, 0x0f, 0xe1, 0x6f, 0xdb, 0xcc, 0xf1, 0x91, 0x7c, 0xf7, 0xd3,
		0xb9, 0xc2, 0x90, 0xb0, 0xc7, 0xf2, 0x43, 0x72, 0x75, 0x38, 0x43, 0x72, 0x52, 0xe3, 0xc0, 0xed,
		0x4c, 0xe3, 0xb4, 0xca, 0x6d, 0x8e, 0xc9, 0x8b, 0xd3, 0xb9, 0xc2, 0x56, 0x77, 0x95, 0x5c, 0x3a,
		0x3e, 0x4a, 0x8f, 0xb5, 0x02, 0xe1, 0x4b, 0x36, 0x70, 0x88, 0xce, 0xdd, 0x43, 0x1b, 0x3d, 0x14,
		0xc3, 0x46, 0x9c, 0xf0, 0x3c, 0x72, 0x7c, 0xff, 0xf4, 0x1e, 0xae, 0xdb, 0xdd, 0xb6, 0xa1, 0xa0,
		0x3a, 0x8a, 0xf1, 0x3a, 0x2b, 0x6e, 0x2f, 0xfa, 0xa6, 0xa8, 0x07, 0x65, 0x43, 0x78, 0x66, 0x28,
		0x4f, 0x3b, 0xbf, 0x6f, 0xda, 0x74, 0x52, 0x42, 0xcf, 0xe2, 0x8c, 0x76, 0xa0, 0x98, 0x01, 0x0f,
		0x14, 0x33, 0xe0, 0x3f, 0x5c, 0x47, 0xb2, 0x11, 0x96, 0xce, 0x80, 0x75, 0x56, 0x7a, 0xa8, 0x59,
		0xe9, 0x17, 0x26, 0x2c, 0x26, 0x5d, 0x6f, 0x56, 0x9c, 0x85, 0x95, 0xc8, 0x60, 0x2d, 0x1c, 0xf2,
		0x09, 0x73, 0xae, 0xbb, 0x0a, 0x59, 0x6c, 0xc7, 0x24, 0xb4, 0xdd, 0x88, 0x3c, 0x57, 0xc7, 0x9b,
		0xfb, 0x5e, 0xbd, 0xb8, 0x44, 0xc4, 0x6c, 0xb7, 0xdb, 0x87, 0x33, 0x2a, 0xfb, 0x1e, 0x3c, 0x86,
		0x8a, 0xc1, 0x63, 0xa8, 0x18, 0x3c, 0xbe, 0x23, 0xb3, 0xc0, 0x15, 0xce, 0x6c, 0x67, 0xe1, 0xc3,
		0xd4, 0xe1, 0xe3, 0x60, 0x41, 0x0d, 0x5f, 0x7a, 0x5c, 0x8c, 0x54, 0xe2, 0xc1, 0x9b, 0xa6, 0x0c,
		0xc3, 0x52, 0x34, 0x0c, 0x4b, 0xd1, 0x30, 0xfa, 0xc2, 0x95, 0x63, 0xf4, 0x20, 0x89, 0x82, 0x3a,
		0xb1, 0xd2, 0x96, 0xa1, 0x93, 0x25, 0x9d, 0x2c, 0xe9, 0x64, 0x69, 0x8f, 0x93, 0x25, 0x54, 0x8c,
		0x09, 0xa8, 0x18, 0x13, 0xa2, 0x14, 0xc9, 0xe1, 0xbe, 0xd4, 0xd1, 0x40, 0x47, 0x83, 0xbc, 0x71,
		0xe6, 0x42, 0x76, 0xae, 0x15, 0x22, 0x81, 0x79, 0xb8, 0x3e, 0xbd, 0xba, 0xff, 0xaa, 0x80, 0x3b,
		0x87, 0x2a, 0xb3, 0x97, 0x4e, 0x5d, 0x63, 0xf1, 0x4a, 0x16, 0xf6, 0x99, 0xfb, 0xb2, 0x2f, 0xa5,
		0x47, 0xb3, 0xb2, 0x2f, 0x5c, 0x7c, 0x74, 0x30, 0xb4, 0x7f, 0xe2, 0x50, 0x85, 0xe2, 0x5c, 0xba,
		0xa3, 0xf3, 0xa6, 0xdb, 0xbd, 0x7e, 0xdd, 0xed, 0xb6, 0x5f, 0x5f, 0xbd, 0x6e, 0xbf, 0xed, 0xf5,
		0x3a, 0xd7, 0x1d, 0x4a, 0xf5, 0xf5, 0x37, 0xcf, 0x42, 0x0f, 0xad, 0x77, 0x33, 0xe3, 0x06, 0x44,
		0xe0, 0x38, 0x4d, 0x45, 0x31, 0x5b, 0x31, 0x8a, 0xd9, 0x8a, 0x51, 0x6c, 0xc0, 0x05, 0xf3, 0x56,
		0xe7, 0xfb, 0x43, 0x1d, 0xc7, 0x74, 0x1c, 0xcb, 0x18, 0xe7, 0x58, 0x55, 0x14, 0x02, 0xd9, 0x5b,
		0x42, 0xd3, 0xcf, 0x28, 0x46, 0x72, 0xbc, 0x77, 0x91, 0xcc, 0x6c, 0xeb, 0xa2, 0xf2, 0x21, 0x8f,
		0xc9, 0xbe, 0x4f, 0x4e, 0x46, 0x8a, 0x6e, 0x7d, 0xa4, 0xe8, 0xd6, 0xdf, 0xb9, 0xae, 0x83, 0x4c,
		0xe8, 0x32, 0xa0, 0xf6, 0xeb, 0xc5, 0x7e, 0x3d, 0xd6, 0x15, 0x15, 0xac, 0xaa, 0x53, 0xd6, 0x2e,
		0x94, 0x38, 0xa5, 0x7d, 0x21, 0x5c, 0xc9, 0x12, 0x95, 0xce, 0xa1, 0x96, 0xfa, 0xc3, 0x31, 0x4e,
		0xd8, 0x94, 0x45, 0x71, 0xc4, 0xb8, 0x74, 0x85, 0xdd, 0x92, 0xe8, 0xcb, 0xce, 0x65, 0xcc, 0x00,
		0xbf, 0xcc, 0x65, 0x18, 0xc7, 0x4f, 0x90, 0x5e, 0x30, 0x94, 0x22, 0x19, 0x90, 0xdf, 0x84, 0x7d,
		0x1b, 0xde, 0xff, 0xff, 0xd0, 0x62, 0x3a, 0xfd, 0xe8, 0x1f, 0xb3, 0x9f, 0x2d, 0xb8, 0xcd, 0x1e,
		0x65, 0xf4, 0x26, 0xb2, 0xe1, 0x0e, 0x81, 0x0b, 0x9d, 0xb4, 0xa3, 0x71, 0xa1, 0x3f, 0x67, 0xda,
		0xf6, 0xf6, 0xdb, 0xf3, 0x6d, 0x5a, 0x93, 0xa1, 0xeb, 0x23, 0x43, 0x17, 0xda, 0x20, 0xbd, 0x86,
		0xb2, 0xa8, 0x9d, 0xe4, 0xb4, 0x21, 0x26, 0x51, 0xb4, 0xd9, 0x14, 0xdd, 0x95, 0xce, 0x13, 0x83,
		0x1e, 0xd1, 0x11, 0x96, 0xcd, 0x07, 0xd4, 0xf3, 0x80, 0x27, 0xda, 0x34, 0x50, 0xbd, 0xab, 0x9d,
		0xf6, 0xfe, 0xf5, 0xb5, 0xa4, 0x2f, 0xbe, 0xab, 0xe2, 0xcf, 0xb8, 0x4f, 0x5a, 0xdb, 0x91, 0xb4,
		0xa3, 0xf9, 0xb3, 0x3e, 0xf8, 0x7c, 0x32, 0x75, 0x30, 0x06, 0x55, 0x5d, 0x3b, 0x9c, 0x88, 0xda,
		0x7c, 0x14, 0x78, 0x51, 0x08, 0x00, 0x2e, 0x71, 0xe2, 0xeb, 0x85, 0x1e, 0x7b, 0xbf, 0xd0, 0x23,
		0x89, 0xa2, 0xc4, 0xec, 0x36, 0x6a, 0xad, 0x96, 0xdb, 0xde, 0x8e, 0x13, 0x15, 0xe1, 0x3e, 0xdc,
		0xe3, 0x0c, 0x2d, 0x18, 0xcc, 0x80, 0xf2, 0x1c, 0x9d, 0xd4, 0x1e, 0x4f, 0x52, 0x5b, 0x82, 0x9c,
		0x70, 0xb8, 0x68, 0x45, 0x57, 0x83, 0x15, 0xeb, 0x43, 0xf2, 0x46, 0x63, 0x15, 0x35, 0x60, 0x15,
		0x1e, 0x13, 0x23, 0x6c, 0x4d, 0x08, 0x82, 0x48, 0x0d, 0x6f, 0x71, 0x8b, 0x22, 0xc5, 0x06, 0x26,
		0xec, 0x11, 0x1e, 0x42, 0xf1, 0x81, 0xed, 0x7a, 0x20, 0xc7, 0x08, 0xd1, 0xb3, 0xb4, 0x57, 0xd7,
		0x5e, 0xfd, 0xf8, 0xd6, 0xd1, 0x69, 0xfc, 0x79, 0x13, 0x7f, 0xee, 0xf5, 0xb4, 0x53, 0xaf, 0xcf,
		0xa9, 0x13, 0xb4, 0x73, 0xdd, 0xa9, 0x73, 0xa1, 0xec, 0xd4, 0x93, 0x29, 0x5d, 0xf4, 0x00, 0x90,
		0x2e, 0x48, 0xf4, 0x25, 0x78, 0x81, 0x83, 0x3e, 0x70, 0x01, 0xff, 0xe9, 0x7f, 0xfd, 0xd7, 0x05,
		0x7c, 0xe1, 0x02, 0x26, 0x81, 0x2f, 0x61, 0x80, 0xf0, 0xbf, 0xa0, 0xdd, 0xbe, 0x1a, 0xfe, 0x02,
		0x84, 0x00, 0xa2, 0x1d, 0xff, 0xa1, 0x3a, 0xfe, 0x66, 0x97, 0xaa, 0xe8, 0x20, 0xa1, 0x83, 0x84,
		0x0e, 0x12, 0x55, 0x83, 0x04, 0x2a, 0xb1, 0xea, 0xe3, 0xe6, 0xaa, 0xc1, 0x21, 0x5d, 0x65, 0x1a,
		0x46, 0x86, 0x98, 0x98, 0x1f, 0x46, 0x85, 0x30, 0xf5, 0x37, 0x59, 0xe1, 0xee, 0x2e, 0x3a, 0x10,
		0x1c, 0xe1, 0x0c, 0x20, 0xd4, 0x12, 0x0f, 0x6d, 0x15, 0x60, 0xe7, 0x35, 0xa1, 0xed, 0xb7, 0x79,
		0xf1, 0x70, 0xa5, 0x64, 0x78, 0x99, 0x2c, 0x16, 0x69, 0xc0, 0xbe, 0xe4, 0x63, 0x6b, 0xea, 0xfe,
		0x44, 0x8f, 0x6e, 0x62, 0xe9, 0x1d, 0x8a, 0x68, 0xa9, 0xc7, 0x84, 0x3f, 0xe1, 0x12, 0x48, 0x37,
		0x6b, 0x53, 0x3a, 0xae, 0xc9, 0xb4, 0x12, 0x31, 0xf9, 0x5a, 0xef, 0x4a, 0xf3, 0x82, 0x12, 0x25,
		0xcd, 0xe6, 0x22, 0xfb, 0x71, 0xa5, 0x02, 0xd9, 0xaf, 0x38, 0x2b, 0x28, 0x6c, 0xd1, 0x18, 0xce,
		0x74, 0x66, 0xf3, 0x1a, 0xa3, 0x39, 0xa7, 0x1a, 0x40, 0xa3, 0x2d, 0x6f, 0xeb, 0x99, 0xc2, 0x26,
		0x29, 0x46, 0x38, 0x9f, 0xaf, 0x83, 0x90, 0x90, 0xd6, 0x09, 0x83, 0xc9, 0x00, 0xbd, 0xd3, 0x8b,
		0x4b, 0xd9, 0xb9, 0x49, 0xd1, 0x88, 0xb3, 0x14, 0x2e, 0xc8, 0xf8, 0x9a, 0x3d, 0x9e, 0x51, 0x7c,
		0xdb, 0x6a, 0xb8, 0x24, 0x06, 0x93, 0x34, 0x54, 0x8d, 0xb9, 0x0f, 0xdc, 0x07, 0x16, 0xe3, 0x17,
		0xbe, 0x64, 0x32, 0x12, 0x03, 0x35, 0xb6, 0x94, 0xd8, 0xa9, 0x65, 0x39, 0x90, 0x59, 0x4b, 0xef,
		0xae, 0xe0, 0x37, 0xaa, 0xec, 0xd1, 0xb2, 0x1a, 0xd5, 0xb6, 0x75, 0xbf, 0x26, 0x83, 0xa5, 0xd0,
		0x38, 0x3e, 0x7a, 0x9e, 0xeb, 0xf5, 0xa7, 0xd3, 0x5b, 0x36, 0x52, 0x97, 0x5f, 0x0c, 0x4b, 0x45,
		0xba, 0xba, 0x1b, 0x89, 0x61, 0xf8, 0xb6, 0x2d, 0x36, 0x9d, 0xb6, 0x24, 0x1b, 0x3d, 0x8b, 0xcc,
		0x96, 0xba, 0xbc, 0x6b, 0x29, 0x7d, 0x41, 0xdf, 0x67, 0x23, 0x2c, 0x29, 0xa6, 0xd0, 0xe0, 0x53,
		0x98, 0xd0, 0x41, 0xdf, 0x07, 0x39, 0x66, 0x02, 0x5c, 0x0f, 0xf0, 0xcf, 0x80, 0x39, 0xe1, 0x14,
		0x92, 0x5a, 0x7b, 0xaa, 0x57, 0x9a, 0x93, 0xa4, 0x5b, 0xcf, 0x26, 0x4d, 0xa5, 0x91, 0xa9, 0x4b,
		0xe8, 0xf5, 0xb2, 0x8e, 0x1a, 0x66, 0x80, 0xe6, 0xf2, 0x90, 0xa0, 0x98, 0x01, 0x1a, 0x06, 0xec,
		0x8a, 0x0c, 0x50, 0xee, 0xcb, 0x2e, 0x8d, 0x30, 0xd5, 0x25, 0xf3, 0xa5, 0x22, 0x16, 0xcc, 0x4f,
		0x2e, 0xc7, 0xc0, 0x20, 0x99, 0x18, 0x03, 0x17, 0x16, 0x3e, 0xee, 0x07, 0x4d, 0x0a, 0x0f, 0x91,
		0x27, 0x85, 0x3b, 0x23, 0x4a, 0x71, 0x05, 0x7c, 0x8d, 0xab, 0x82, 0x6b, 0x9f, 0xb9, 0xb8, 0x8f,
		0x50, 0xb5, 0x48, 0xf3, 0x23, 0x7a, 0x94, 0x7f, 0x18, 0x93, 0x7f, 0x7c, 0x89, 0xb3, 0x7f, 0x3c,
		0x3e, 0x24, 0x4d, 0x76, 0x6e, 0x12, 0xef, 0x2b, 0x3b, 0x37, 0xb1, 0x1a, 0x86, 0x9f, 0x22, 0x17,
		0xdb, 0xd0, 0xb2, 0x9a, 0xae, 0xe2, 0xee, 0x7a, 0xdd, 0x41, 0x99, 0x35, 0xff, 0x5d, 0x06, 0xae,
		0x88, 0x0c, 0xab, 0xcb, 0x00, 0xe7, 0xf3, 0x2e, 0x6d, 0x5b, 0xda, 0xb6, 0x8e, 0x93, 0x7d, 0xa8,
		0xc1, 0x35, 0x0d, 0xae, 0x35, 0xb4, 0x54, 0x32, 0xf2, 0xb1, 0x0a, 0x3e, 0x3d, 0x6e, 0xaf, 0x5a,
		0x86, 0x4c, 0xd3, 0x68, 0x2e, 0xc2, 0x44, 0x3a, 0x4d, 0xaa, 0x4d, 0xb8, 0xc7, 0x99, 0x0f, 0xcc,
		0x8f, 0x97, 0x51, 0x7a, 0x68, 0x53, 0xdd, 0x7c, 0x47, 0xbb, 0xf9, 0xc3, 0x73, 0xf3, 0x45, 0x09,
		0x7b, 0xda, 0xd0, 0xe2, 0xfe, 0xd4, 0x61, 0x33, 0xd2, 0x42, 0x87, 0x0d, 0xe9, 0x2c, 0xdf, 0x4c,
		0x1c, 0x87, 0x35, 0x85, 0x0d, 0xa7, 0xf6, 0xd1, 0x1f, 0xcc, 0x81, 0xe4, 0x69, 0x51, 0x7e, 0x0f,
		0x4c, 0x4a, 0x8f, 0x0f, 0x02, 0x89, 0x73, 0xf5, 0xb5, 0xb8, 0x1d, 0x15, 0xd8, 0x25, 0x38, 0x51,
		0xe8, 0x88, 0x61, 0x00, 0x9f, 0xfa, 0xbb, 0xb4, 0x9c, 0x45, 0x59, 0xa9, 0xcb, 0x28, 0x77, 0x49,
		0x25, 0xaf, 0x02, 0xf4, 0x54, 0x52, 0xfa, 0x5a, 0x90, 0x9e, 0x72, 0x46, 0xa0, 0xe8, 0xd8, 0x89,
		0xb2, 0x22, 0xe7, 0x40, 0xe5, 0x73, 0xa1, 0x32, 0x39, 0x51, 0xd9, 0xdc, 0x48, 0x3d, 0x47, 0xaa,
		0x94, 0x2b, 0x55, 0xcd, 0x99, 0x6a, 0xcb, 0x13, 0xaa, 0xe7, 0x0b, 0x25, 0x72, 0xa9, 0x4a, 0x39,
		0xd5, 0xc6, 0xd0, 0xf5, 0x0e, 0x7f, 0xe8, 0x4e, 0x1a, 0x1c, 0xe8, 0x67, 0xd3, 0xe8, 0xb6, 0x56,
		0xe9, 0xb2, 0x63, 0x67, 0xb6, 0x8f, 0x4d, 0xa7, 0xc9, 0xad, 0xef, 0x76, 0x58, 0xa4, 0xb2, 0xef,
		0x71, 0xd6, 0x51, 0xcf, 0xe8, 0xe2, 0xdb, 0xca, 0xe5, 0x72, 0xb6, 0xeb, 0x21, 0x1f, 0x89, 0x70,
		0xa2, 0x01, 0x1d, 0x68, 0x85, 0x73, 0x8c, 0x39, 0x6a, 0xdb, 0x63, 0x97, 0x2a, 0x0f, 0xd6, 0xc9,
		0x9a, 0x4e, 0xd6, 0xaa, 0x27, 0x6b, 0x74, 0x50, 0xb8, 0x0c, 0x38, 0x9c, 0x0f, 0x12, 0x63, 0x84,
		0x12, 0xf7, 0xa2, 0x4f, 0x91, 0xe6, 0xef, 0xd8, 0xf4, 0xcd, 0x72, 0xa6, 0x6f, 0xd6, 0x60, 0xfa,
		0x66, 0x96, 0xe9, 0x9b, 0xda, 0xf4, 0xb5, 0xe9, 0x1f, 0xa5, 0xe9, 0x9b, 0x3b, 0x62, 0x29, 0x9c,
		0x17, 0x96, 0x9e, 0xd3, 0xf8, 0x0e, 0x14, 0x53, 0x7f, 0x89, 0x5b, 0x97, 0x36, 0x70, 0x70, 0x1a,
		0x89, 0x1b, 0xa8, 0x3e, 0x9f, 0x59, 0x3e, 0x51, 0x78, 0xc8, 0xe4, 0xe9, 0xab, 0x28, 0xbd, 0x86,
		0x57, 0xe7, 0x10, 0x43, 0x10, 0xa7, 0x17, 0x17, 0x97, 0xff, 0x0c, 0x35, 0x8c, 0x5b, 0x67, 0xe7,
		0xf0, 0xaa, 0xb5, 0xf4, 0x45, 0x7c, 0x3d, 0x12, 0xf4, 0xf6, 0xaf, 0xcc, 0xb3, 0x33, 0xf8, 0x65,
		0x71, 0x3d, 0xbc, 0xbc, 0x04, 0xe6, 0x9d, 0xa9, 0x18, 0x4e, 0x19, 0xd2, 0xe1, 0xa6, 0x6b, 0x4d,
		0xd9, 0x77, 0x3e, 0x8b, 0x16, 0x59, 0x96, 0xe2, 0x20, 0x56, 0x75, 0xbd, 0x50, 0x0b, 0x27, 0xb1,
		0x36, 0x3f, 0x0c, 0xdb, 0x39, 0x8a, 0xd9, 0xa3, 0xd4, 0xd0, 0xbc, 0x45, 0xa5, 0xd4, 0x54, 0x86,
		0xc2, 0x98, 0x87, 0x2b, 0x83, 0xed, 0x7a, 0x13, 0xf6, 0x5c, 0x0a, 0x50, 0x96, 0xe2, 0xd8, 0x9c,
		0x0a, 0x64, 0x0c, 0xcd, 0xbe, 0x48, 0x5d, 0x95, 0x12, 0x99, 0x2b, 0xf6, 0x39, 0x05, 0x30, 0xee,
		0xa3, 0x44, 0x2b, 0x71, 0x56, 0xe0, 0xf0, 0x7b, 0x84, 0xc4, 0x1d, 0xc6, 0x8c, 0xe9, 0x8b, 0x8b,
		0x4b, 0x6e, 0x45, 0x1f, 0xb1, 0x15, 0x5f, 0x89, 0x9c, 0xdf, 0xc6, 0x15, 0x33, 0xbe, 0xf2, 0xea,
		0x59, 0x75, 0x49, 0x9d, 0x60, 0xb9, 0x1b, 0x5d, 0x6a, 0x72, 0xbc, 0x9b, 0x52, 0xd0, 0xbd, 0xd8,
		0xe1, 0x9d, 0x48, 0xeb, 0x4c, 0xdb, 0x53, 0xe8, 0x9d, 0xdd, 0x4b, 0x52, 0xf5, 0x17, 0x68, 0x64,
		0xcf, 0x6e, 0xfc, 0xff, 0xbe, 0xb1, 0xcb, 0x25, 0x23, 0x5b, 0x49, 0x7e, 0x0d, 0x2f, 0x18, 0x51,
		0xcb, 0x23, 0xab, 0xad, 0x21, 0xa9, 0x93, 0xd1, 0xdb, 0xad, 0x46, 0xe8, 0xed, 0x56, 0xe5, 0xf3,
		0xf6, 0x68, 0x7c, 0xde, 0x5e, 0x09, 0x3e, 0x6f, 0x4c, 0x3d, 0xd0, 0x34, 0xde, 0x03, 0xa0, 0xf1,
		0x92, 0x40, 0xe3, 0x74, 0x38, 0x09, 0x90, 0xee, 0xba, 0x5e, 0x44, 0xd3, 0x4f, 0x37, 0x39, 0x6b,
		0xa8, 0x07, 0x2d, 0x20, 0x55, 0x54, 0x35, 0xe3, 0xb0, 0xc6, 0x74, 0x40, 0x33, 0x0e, 0x35, 0xe3,
		0x50, 0x33, 0x0e, 0xeb, 0x1a, 0x93, 0x3d, 0x66, 0x1c, 0x92, 0xca, 0x00, 0xcb, 0xce, 0xdc, 0x54,
		0x77, 0xe6, 0xe6, 0x8a, 0x33, 0x8f, 0x57, 0xae, 0x6a, 0x67, 0xae, 0x9d, 0xf9, 0x31, 0xee, 0x60,
		0x65, 0x6a, 0x4f, 0xbe, 0x11, 0xdc, 0xb4, 0x27, 0xaf, 0xc3, 0x93, 0x3b, 0xc8, 0xec, 0x1e, 0x53,
		0x5b, 0x0f, 0xd4, 0x53, 0xe5, 0x8e, 0x0b, 0x37, 0xae, 0xdf, 0x2e, 0x98, 0xb7, 0x3a, 0x4f, 0xd7,
		0xae, 0x5d, 0xe7, 0xe9, 0x3a, 0x4f, 0xd7, 0x79, 0x7a, 0x75, 0xef, 0x5e, 0x06, 0x43, 0x8d, 0xc0,
		0x92, 0x9c, 0xbc, 0xfc, 0x05, 0x41, 0xa9, 0xcf, 0xb9, 0x1d, 0xcf, 0xbc, 0xc4, 0xde, 0x63, 0xb0,
		0x51, 0x2b, 0xdf, 0x5e, 0x45, 0xcf, 0x2e, 0xa2, 0xc7, 0x91, 0x77, 0x97, 0x9b, 0xf5, 0x54, 0xa9,
		0x97, 0xbf, 0x9c, 0x3d, 0x7b, 0x2a, 0xd4, 0xc3, 0xf7, 0x60, 0xeb, 0x9e, 0x58, 0x6b, 0xd4, 0x4a,
		0xdd, 0x87, 0xbf, 0x7b, 0xcf, 0x6a, 0xaf, 0x0f, 0x66, 0x03, 0x9f, 0xe4, 0xb5, 0x0b, 0x0a, 0xa6,
		0x3d, 0x96, 0x54, 0x4b, 0x37, 0x4a, 0xa3, 0xea, 0x95, 0xe8, 0xc3, 0xdf, 0xda, 0xa7, 0xce, 0x31,
		0x3b, 0xda, 0x4d, 0x7f, 0x7a, 0xd5, 0x4a, 0x84, 0x3d, 0x72, 0x89, 0xf0, 0x24, 0xa7, 0x67, 0x46,
		0x3f, 0x18, 0x85, 0x52, 0x45, 0x2b, 0x33, 0x04, 0x17, 0x94, 0x10, 0x17, 0x44, 0x49, 0x6a, 0x1d,
		0xd1, 0xb2, 0x80, 0xc1, 0xd0, 0x0d, 0x42, 0xef, 0xee, 0xda, 0x20, 0xf0, 0x67, 0x34, 0x27, 0xf5,
		0x41, 0xba, 0x90, 0xff, 0x24, 0x5d, 0x58, 0xac, 0x32, 0xa7, 0xac, 0xb9, 0xb0, 0x98, 0xbf, 0x65,
		0xd4, 0xa6, 0x8f, 0x2d, 0xa8, 0x88, 0x43, 0x2d, 0x5b, 0x48, 0x11, 0x55, 0x45, 0xc3, 0x16, 0xfb,
		0x0c, 0x5b, 0x14, 0xf1, 0x4d, 0xd4, 0x26, 0x4b, 0xea, 0x93, 0xa6, 0x5a, 0x26, 0x4f, 0x6a, 0x93,
		0xa8, 0x8a, 0x60, 0x62, 0x2e, 0xdb, 0x23, 0xd3, 0x14, 0x7b, 0x15, 0x4c, 0x31, 0x97, 0xfd, 0xa1,
		0x2d, 0xf0, 0xe5, 0x58, 0x60, 0x11, 0x5a, 0xf1, 0x92, 0x0d, 0xf1, 0x80, 0x17, 0x16, 0xec, 0x16,
		0xf5, 0xa8, 0x84, 0x7e, 0x34, 0x84, 0x82, 0x54, 0x99, 0x66, 0xd5, 0x87, 0x8a, 0xd4, 0x32, 0xe9,
		0x6a, 0x0e, 0x25, 0x51, 0x03, 0x52, 0xe1, 0x39, 0x56, 0x0d, 0x94, 0x41, 0x51, 0xea, 0x96, 0xfd,
		0xfe, 0x2d, 0x18, 0x28, 0x81, 0xb2, 0xec, 0x40, 0xd6, 0x95, 0xd7, 0x0a, 0xec, 0x1e, 0x85, 0x69,
		0x46, 0x55, 0xf6, 0x69, 0x3d, 0xc0, 0xce, 0x51, 0x9a, 0x12, 0x8a, 0xb6, 0x9f, 0x75, 0x9c, 0x02,
		0x0c, 0xe5, 0x6e, 0x0d, 0x43, 0x21, 0xe4, 0x03, 0xf9, 0xf1, 0x9f, 0x00, 0x49, 0x08, 0x57, 0x9e,
		0xa6, 0x9b, 0x95, 0xfe, 0xf0, 0x51, 0xb6, 0x92, 0x33, 0xa8, 0xfc, 0x53, 0xdb, 0x75, 0x1c, 0xf7,
		0x27, 0x17, 0xa3, 0x96, 0xcf, 0x07, 0x0e, 0x17, 0xa3, 0x9b, 0x9b, 0x95, 0x6d, 0x4d, 0xe7, 0xa7,
		0xf3, 0x9c, 0xc3, 0xd2, 0x1f, 0x67, 0x77, 0x79, 0xf1, 0x5d, 0x29, 0x9e, 0xa7, 0xef, 0xf8, 0xc0,
		0x1c, 0x6e, 0xc5, 0x27, 0xe2, 0xdb, 0x8c, 0x3b, 0x3e, 0x70, 0x1b, 0xe6, 0xbf, 0x17, 0x46, 0x2c,
		0xe1, 0x4a, 0x08, 0x04, 0xff, 0x33, 0xc0, 0xf9, 0x96, 0x76, 0x85, 0x9b, 0x5f, 0xab, 0x5a, 0x68,
		0xf9, 0xc0, 0x5d, 0xca, 0xfe, 0x56, 0xec, 0xad, 0x74, 0xf7, 0x1b, 0x98, 0x15, 0xaa, 0x44, 0xe0,
		0x95, 0xb9, 0xa1, 0xc9, 0x28, 0x67, 0x0e, 0x94, 0x15, 0x89, 0x6a, 0x3c, 0xad, 0x2e, 0x94, 0xe5,
		0x3e, 0x35, 0x35, 0xce, 0xd4, 0xe8, 0xb7, 0x71, 0x5e, 0x56, 0xec, 0x9b, 0x43, 0xbd, 0x18, 0x20,
		0x78, 0x38, 0x45, 0x16, 0xba, 0xe7, 0xa6, 0xed, 0x42, 0x35, 0x52, 0x55, 0x17, 0x02, 0xb5, 0xbf,
		0x0d, 0x7b, 0xf1, 0xbb, 0x8c, 0xc5, 0x32, 0x53, 0x0f, 0x7d, 0x14, 0x43, 0xac, 0xe2, 0x9b, 0x6f,
		0xdd, 0x29, 0x38, 0xf8, 0x80, 0x4e, 0x68, 0xe5, 0xf1, 0x03, 0x65, 0xfd, 0xf8, 0x71, 0xfc, 0x9a,
		0x4d, 0x22, 0xc8, 0x99, 0xfd, 0xa8, 0x3e, 0xea, 0xf9, 0xf5, 0x87, 0xfc, 0x8a, 0x4a, 0x51, 0x25,
		0x25, 0x63, 0x3c, 0xf2, 0x6b, 0x27, 0xab, 0xfd, 0x59, 0xbc, 0xed, 0xd2, 0x7b, 0x19, 0xd1, 0xa3,
		0x07, 0xad, 0x68, 0x4a, 0xb5, 0xf1, 0x56, 0xcb, 0x33, 0xee, 0x45, 0xab, 0xb5, 0x5e, 0x6d, 0x9e,
		0x35, 0x8c, 0x43, 0x57, 0x58, 0x20, 0xd3, 0x11, 0x4e, 0x4f, 0x90, 0x84, 0x16, 0x44, 0x33, 0x3a,
		0x57, 0xc4, 0x87, 0xcb, 0x47, 0xcf, 0x5b, 0xb0, 0xf6, 0xfc, 0x0b, 0xf8, 0x68, 0x71, 0x09, 0xfe,
		0x6c, 0x32, 0x70, 0x1d, 0xf0, 0xc7, 0x6e, 0xe0, 0x58, 0x73, 0x13, 0x7a, 0xe0, 0x61, 0xc4, 0xdf,
		0xf8, 0xf5, 0x6c, 0xf4, 0x6d, 0x81, 0xb6, 0xad, 0x31, 0x5b, 0xf3, 0xd0, 0xb5, 0xfc, 0x93, 0xef,
		0x8a, 0x34, 0x99, 0x0c, 0x96, 0x91, 0xf5, 0xb6, 0xf0, 0xe4, 0xba, 0xfc, 0x6a, 0xd7, 0xb6, 0xc2,
		0x86, 0x91, 0x9c, 0xa6, 0x58, 0xbc, 0x68, 0x2e, 0x6e, 0x47, 0x5d, 0x35, 0xb7, 0x26, 0xcc, 0x82,
		0xda, 0x56, 0xbb, 0xa8, 0xb6, 0x65, 0xd6, 0x52, 0xdb, 0x3a, 0xc4, 0xd2, 0x56, 0x5d, 0x95, 0xad,
		0x42, 0x56, 0x24, 0xfd, 0x08, 0x42, 0xc2, 0xd1, 0x83, 0x44, 0x66, 0x3b, 0x0d, 0x36, 0xa5, 0x43,
		0xd4, 0x0b, 0xba, 0x76, 0xbb, 0x4d, 0xdd, 0x48, 0xa9, 0x2c, 0x97, 0x4f, 0x9d, 0xc3, 0xf7, 0x44,
		0xc3, 0x7c, 0xd5, 0xbb, 0x6b, 0xee, 0x67, 0x77, 0xeb, 0x25, 0x26, 0x90, 0x57, 0x00, 0x9b, 0x03,
		0xda, 0x12, 0x60, 0x73, 0x40, 0xf7, 0x66, 0xf1, 0x81, 0xf9, 0x51, 0x31, 0xc8, 0xb5, 0x13, 0xe7,
		0xc6, 0x25, 0x4e, 0x2a, 0xaf, 0x06, 0xd6, 0x8e, 0xad, 0xaa, 0x63, 0x2b, 0x3e, 0xd2, 0x27, 0xaa,
		0x9c, 0xd3, 0x4f, 0xf5, 0xa1, 0x14, 0xda, 0xd7, 0xcf, 0xf3, 0x1d, 0x27, 0xca, 0xa1, 0x52, 0xa5,
		0xd7, 0xe7, 0xf9, 0xd6, 0x5e, 0x22, 0xd4, 0xab, 0xc6, 0x2a, 0x95, 0xd4, 0x2a, 0xad, 0x2b, 0x68,
		0xeb, 0x75, 0x05, 0x1b, 0x71, 0xb9, 0xd7, 0xd3, 0x0b, 0x0b, 0x6a, 0xc0, 0x9a, 0xc2, 0xb9, 0xc7,
		0xd5, 0x50, 0x6d, 0xd9, 0xd8, 0xd5, 0x50, 0x99, 0xeb, 0x91, 0x54, 0x07, 0x16, 0xeb, 0xc6, 0xb8,
		0x00, 0x99, 0x78, 0xf6, 0xba, 0x7d, 0xba, 0xa9, 0x7d, 0xfa, 0xc1, 0xfa, 0x74, 0xbd, 0x5c, 0xec,
		0xc8, 0xdd, 0xba, 0x5e, 0x2e, 0x56, 0x11, 0xa0, 0x2e, 0xd8, 0x72, 0x2b, 0xef, 0xb4, 0x54, 0xbd,
		0xeb, 0xd6, 0x2a, 0x10, 0x9c, 0xe0, 0xb0, 0x97, 0xb9, 0x33, 0xda, 0x42, 0x74, 0xf8, 0xdd, 0xef,
		0xe1, 0x43, 0xe2, 0x23, 0x75, 0xdf, 0xd5, 0xc4, 0xaf, 0xaf, 0x84, 0x6f, 0x67, 0xc3, 0xcb, 0xc4,
		0x7e, 0x50, 0xa0, 0xee, 0x30, 0x47, 0xe8, 0xcb, 0x5b, 0x77, 0xfa, 0x19, 0x1f, 0xd0, 0xd9, 0x0e,
		0x76, 0xaf, 0xb5, 0x2b, 0x82, 0xbb, 0xa3, 0x43, 0xc9, 0x98, 0x8c, 0xf2, 0x86, 0x05, 0xe6, 0x7d,
		0x2a, 0x5c, 0x09, 0x1e, 0x0e, 0xdd, 0xc9, 0x04, 0x85, 0x85, 0x16, 0x0c, 0x02, 0x99, 0xd2, 0x13,
		0xfc, 0x60, 0x3a, 0x75, 0x3d, 0x89, 0xd6, 0xd9, 0x16, 0x38, 0xbb, 0xbd, 0x0d, 0xce, 0x6e, 0x1f,
		0x2d, 0x9c, 0xbd, 0x35, 0x68, 0x17, 0x07, 0xe9, 0xbc, 0xa0, 0x6c, 0x7c, 0x63, 0x52, 0xa2, 0x27,
		0xb6, 0x46, 0x61, 0xe3, 0x47, 0xbf, 0xf5, 0xdf, 0xbb, 0xbf, 0xae, 0x9e, 0x5a, 0x3f, 0xda, 0xad,
		0xb7, 0x77, 0xff, 0x30, 0x8a, 0x8a, 0x3e, 0x27, 0xab, 0x9f, 0x92, 0x7e, 0x6c, 0x33, 0x0e, 0x83,
		0xfb, 0x9f, 0xd8, 0x3d, 0x7e, 0x77, 0xdd, 0xcd, 0xd1, 0x5b, 0x37, 0x18, 0x63, 0xf9, 0xab, 0x15,
		0x93, 0xf8, 0x80, 0x0f, 0x7c, 0x98, 0x18, 0xc1, 0xd3, 0xc9, 0xd3, 0xdf, 0x00, 0x00, 0x00, 0xff,
		0xff, 0x01, 0x00, 0x00, 0xff, 0xff, 0x8d, 0xf8, 0x5b, 0x38, 0xe1, 0xd3, 0x00, 0x00,
	}
)
