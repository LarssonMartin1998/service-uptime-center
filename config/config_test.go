package config

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type NestedStruct struct {
	ID       int               `toml:"id"`
	Tags     []string          `toml:"tags"`
	Metadata map[string]string `toml:"metadata"`
}

type TestFoo struct {
	Var1 bool   `toml:"var1"`
	Var2 string `toml:"var2"`
	Var3 int    `toml:"var3"`

	FloatVal float64       `toml:"float_val"`
	Duration time.Duration `toml:"duration"`

	StringSlice []string          `toml:"string_slice"`
	IntSlice    []int             `toml:"int_slice"`
	StringMap   map[string]string `toml:"string_map"`
	IntMap      map[string]int    `toml:"int_map"`

	Nested  NestedStruct            `toml:"nested"`
	Items   []NestedStruct          `toml:"items"`
	Configs map[string]NestedStruct `toml:"configs"`
}

func (t TestFoo) Validate() error {
	if t.Var2 == "" {
		return errEmptyString
	}
	if t.Var3 < 0 {
		return errNegativeInt
	}
	if len(t.StringSlice) == 0 {
		return errEmptySlice
	}
	if len(t.StringMap) == 0 {
		return errEmptyMap
	}
	if t.Nested.ID <= 0 {
		return errInvalidNestedID
	}
	if len(t.Items) == 0 {
		return errNoItems
	}
	return nil
}

var (
	errEmptyString     = errors.New("empty string not allowed")
	errNegativeInt     = errors.New("negative integer not allowed")
	errEmptySlice      = errors.New("string slice cannot be empty")
	errEmptyMap        = errors.New("string map cannot be empty")
	errInvalidNestedID = errors.New("nested ID must be positive")
	errNoItems         = errors.New("items slice cannot be empty")
)

var (
	testConfigToml = `
var1 = true
var2 = "test-value"
var3 = 42
float_val = 3.14
duration = "5m30s"

string_slice = ["apple", "banana", "cherry"]
int_slice = [1, 2, 3, 4, 5]

[string_map]
key1 = "value1"
key2 = "value2"
key3 = "value3"

[int_map]
count1 = 10
count2 = 20
count3 = 30

[nested]
id = 100
tags = ["tag1", "tag2"]

[nested.metadata]
author = "test-author"
version = "1.0.0"

[[items]]
id = 1
tags = ["item1-tag"]

[items.metadata]
type = "first"

[[items]]
id = 2
tags = ["item2-tag1", "item2-tag2"]

[items.metadata]
type = "second"
priority = "high"

[configs.primary]
id = 500
tags = ["primary", "main"]

[configs.primary.metadata]
env = "production"

[configs.secondary]
id = 600
tags = ["secondary", "backup"]

[configs.secondary.metadata]
env = "staging"
`
	expectedResult = TestFoo{
		Var1:     true,
		Var2:     "test-value",
		Var3:     42,
		FloatVal: 3.14,
		Duration: 5*time.Minute + 30*time.Second,

		StringSlice: []string{"apple", "banana", "cherry"},
		IntSlice:    []int{1, 2, 3, 4, 5},

		StringMap: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
		IntMap: map[string]int{
			"count1": 10,
			"count2": 20,
			"count3": 30,
		},

		Nested: NestedStruct{
			ID:   100,
			Tags: []string{"tag1", "tag2"},
			Metadata: map[string]string{
				"author":  "test-author",
				"version": "1.0.0",
			},
		},

		Items: []NestedStruct{
			{
				ID:   1,
				Tags: []string{"item1-tag"},
				Metadata: map[string]string{
					"type": "first",
				},
			},
			{
				ID:   2,
				Tags: []string{"item2-tag1", "item2-tag2"},
				Metadata: map[string]string{
					"type":     "second",
					"priority": "high",
				},
			},
		},

		Configs: map[string]NestedStruct{
			"primary": {
				ID:   500,
				Tags: []string{"primary", "main"},
				Metadata: map[string]string{
					"env": "production",
				},
			},
			"secondary": {
				ID:   600,
				Tags: []string{"secondary", "backup"},
				Metadata: map[string]string{
					"env": "staging",
				},
			},
		},
	}
)

func TestConfigCreation(t *testing.T) {
	cfg, err := Parse(TomlStringDecoder[*TestFoo], testConfigToml)
	if err != nil {
		t.Errorf("failed to create config: %v", err)
	}

	if diff := cmp.Diff(expectedResult, *cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("testConfig should be valid", func(t *testing.T) {
		cfg, err := Parse(TomlStringDecoder[*TestFoo], testConfigToml)
		if err != nil {
			t.Fatalf("testConfig should parse without error: %v", err)
		}

		if err := cfg.Validate(); err != nil {
			t.Fatalf("testConfig should validate without error: %v", err)
		}
	})

	tests := []struct {
		name        string
		config      TestFoo
		expectError error
	}{
		{
			name: "empty string",
			config: TestFoo{
				Var1:        true,
				Var2:        "",
				Var3:        10,
				StringSlice: []string{"test"},
				StringMap:   map[string]string{"key": "value"},
				Nested:      NestedStruct{ID: 1},
				Items:       []NestedStruct{{ID: 1}},
			},
			expectError: errEmptyString,
		},
		{
			name: "negative integer",
			config: TestFoo{
				Var1:        true,
				Var2:        "valid",
				Var3:        -1,
				StringSlice: []string{"test"},
				StringMap:   map[string]string{"key": "value"},
				Nested:      NestedStruct{ID: 1},
				Items:       []NestedStruct{{ID: 1}},
			},
			expectError: errNegativeInt,
		},
		{
			name: "empty slice",
			config: TestFoo{
				Var1:        true,
				Var2:        "valid",
				Var3:        5,
				StringSlice: []string{},
				StringMap:   map[string]string{"key": "value"},
				Nested:      NestedStruct{ID: 1},
				Items:       []NestedStruct{{ID: 1}},
			},
			expectError: errEmptySlice,
		},
		{
			name: "empty map",
			config: TestFoo{
				Var1:        true,
				Var2:        "valid",
				Var3:        5,
				StringSlice: []string{"test"},
				StringMap:   map[string]string{},
				Nested:      NestedStruct{ID: 1},
				Items:       []NestedStruct{{ID: 1}},
			},
			expectError: errEmptyMap,
		},
		{
			name: "invalid nested ID",
			config: TestFoo{
				Var1:        true,
				Var2:        "valid",
				Var3:        5,
				StringSlice: []string{"test"},
				StringMap:   map[string]string{"key": "value"},
				Nested:      NestedStruct{ID: -1},
				Items:       []NestedStruct{{ID: 1}},
			},
			expectError: errInvalidNestedID,
		},
		{
			name: "no items",
			config: TestFoo{
				Var1:        true,
				Var2:        "valid",
				Var3:        5,
				StringSlice: []string{"test"},
				StringMap:   map[string]string{"key": "value"},
				Nested:      NestedStruct{ID: 1},
				Items:       []NestedStruct{},
			},
			expectError: errNoItems,
		},
		{
			name: "valid config",
			config: TestFoo{
				Var1:        false,
				Var2:        "valid",
				Var3:        5,
				StringSlice: []string{"test"},
				StringMap:   map[string]string{"key": "value"},
				Nested:      NestedStruct{ID: 1},
				Items:       []NestedStruct{{ID: 1}},
			},
			expectError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()

			if test.expectError == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %v but got none", test.expectError)
				} else if !errors.Is(err, test.expectError) {
					t.Errorf("expected error %v but got %v", test.expectError, err)
				}
			}
		})
	}
}
