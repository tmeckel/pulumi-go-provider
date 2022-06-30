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

package introspect_test

import (
	"reflect"
	"testing"

	"github.com/pulumi/pulumi-go-provider/internal/introspect"
	"github.com/stretchr/testify/assert"
)

type MyStruct struct {
	Foo  string `pulumi:"foo,optional" provider:"secret,output,description=This is a foo."`
	Bar  int    `provider:"secret"`
	Fizz *int   `pulumi:"fizz"`
}

func TestParseTag(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(MyStruct{})

	cases := []struct {
		Field    string
		Expected introspect.FieldTag
		Error    string
	}{
		{
			Field: "Foo",
			Expected: introspect.FieldTag{
				Name:        "foo",
				Optional:    true,
				Secret:      true,
				Output:      true,
				Description: "This is a foo.",
			},
		},
		{
			Field: "Bar",
			Error: "you must put to the `pulumi` tag to use the `provider` tag",
		},
		{
			Field: "Fizz",
			Expected: introspect.FieldTag{
				Name: "fizz",
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Field, func(t *testing.T) {
			t.Parallel()
			field, ok := typ.FieldByName(c.Field)
			assert.True(t, ok)
			tag, err := introspect.ParseTag(field)
			if c.Error != "" {
				assert.Equal(t, c.Error, err.Error())
			} else {
				assert.Equal(t, c.Expected, tag)
			}
		})
	}
}