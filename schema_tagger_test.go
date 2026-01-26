/*
 * Copyright (C) 2026 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package frags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test basic types
func TestStructToSchema_BasicTypes(t *testing.T) {
	type BasicStruct struct {
		Name   string
		Age    int
		Height float64
		Active bool
		Count  uint
		Score  int32
		Ratio  float32
	}

	schema := StructToSchema(BasicStruct{})

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.Len(t, schema.Properties, 7)
	assert.Len(t, schema.Required, 7)

	assert.Equal(t, "string", schema.Properties["Name"].Type)
	assert.Equal(t, "integer", schema.Properties["Age"].Type)
	assert.Equal(t, "number", schema.Properties["Height"].Type)
	assert.Equal(t, "boolean", schema.Properties["Active"].Type)
	assert.Equal(t, "integer", schema.Properties["Count"].Type)
	assert.Equal(t, "integer", schema.Properties["Score"].Type)
	assert.Equal(t, "number", schema.Properties["Ratio"].Type)
}

// Test JSON tags
func TestStructToSchema_JSONTags(t *testing.T) {
	type TaggedStruct struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Ignored   string `json:"-"`
		Default   string
	}

	schema := StructToSchema(TaggedStruct{})

	assert.NotNil(t, schema)
	assert.Len(t, schema.Properties, 3)

	assert.Contains(t, schema.Properties, "first_name")
	assert.Contains(t, schema.Properties, "last_name")
	assert.Contains(t, schema.Properties, "Default")
	assert.NotContains(t, schema.Properties, "Ignored")

	assert.Contains(t, schema.Required, "first_name")
	assert.Contains(t, schema.Required, "last_name")
	assert.Contains(t, schema.Required, "Default")
	assert.NotContains(t, schema.Required, "Ignored")
}

// Test frags tag - x-session
func TestStructToSchema_FragsXSession(t *testing.T) {
	type SessionStruct struct {
		Token   string  `json:"token" frags:"x-session=auth-token"`
		APIKey  *string `json:"api_key" frags:"x-session=api-credentials"`
		Regular string  `json:"regular"`
	}

	schema := StructToSchema(SessionStruct{})

	assert.NotNil(t, schema)

	// Token should have x-session
	assert.NotNil(t, schema.Properties["token"].XSession)
	assert.Equal(t, "auth-token", *schema.Properties["token"].XSession)

	// APIKey should have x-session
	assert.NotNil(t, schema.Properties["api_key"].XSession)
	assert.Equal(t, "api-credentials", *schema.Properties["api_key"].XSession)

	// Regular should not have x-session
	assert.Nil(t, schema.Properties["regular"].XSession)
}

// Test frags tag - description
func TestStructToSchema_FragsDescription(t *testing.T) {
	type DescribedStruct struct {
		Name  string `json:"name" frags:"description=User's full name"`
		Email string `json:"email" frags:"description=Contact email address"`
		Plain string `json:"plain"`
	}

	schema := StructToSchema(DescribedStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, "User's full name", schema.Properties["name"].Description)
	assert.Equal(t, "Contact email address", schema.Properties["email"].Description)
	assert.Empty(t, schema.Properties["plain"].Description)
}

// Test frags tag - enum
func TestStructToSchema_FragsEnum(t *testing.T) {
	type EnumStruct struct {
		Role   string `json:"role" frags:"enum=admin|user|guest"`
		Status string `json:"status" frags:"enum=active|inactive|pending"`
		Plain  string `json:"plain"`
	}

	schema := StructToSchema(EnumStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, []string{"admin", "user", "guest"}, schema.Properties["role"].Enum)
	assert.Equal(t, []string{"active", "inactive", "pending"}, schema.Properties["status"].Enum)
	assert.Nil(t, schema.Properties["plain"].Enum)
}

// Test frags tag - format
func TestStructToSchema_FragsFormat(t *testing.T) {
	type FormatStruct struct {
		Email     string `json:"email" frags:"format=email"`
		URL       string `json:"url" frags:"format=uri"`
		Timestamp string `json:"timestamp" frags:"format=date-time"`
	}

	schema := StructToSchema(FormatStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, "email", schema.Properties["email"].Format)
	assert.Equal(t, "uri", schema.Properties["url"].Format)
	assert.Equal(t, "date-time", schema.Properties["timestamp"].Format)
}

// Test frags tag - pattern
func TestStructToSchema_FragsPattern(t *testing.T) {
	type PatternStruct struct {
		Phone string `json:"phone" frags:"pattern=^[0-9]{3}-[0-9]{3}-[0-9]{4}$"`
		Code  string `json:"code" frags:"pattern=^[A-Z]{3}$"`
	}

	schema := StructToSchema(PatternStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, "^[0-9]{3}-[0-9]{3}-[0-9]{4}$", schema.Properties["phone"].Pattern)
	assert.Equal(t, "^[A-Z]{3}$", schema.Properties["code"].Pattern)
}

// Test frags tag - title
func TestStructToSchema_FragsTitle(t *testing.T) {
	type TitledStruct struct {
		Name string `json:"name" frags:"title=Full Name"`
		Age  int    `json:"age" frags:"title=Age in Years"`
	}

	schema := StructToSchema(TitledStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, "Full Name", schema.Properties["name"].Title)
	assert.Equal(t, "Age in Years", schema.Properties["age"].Title)
}

// Test multiple frags tags coexisting
func TestStructToSchema_MultipleFragsTags(t *testing.T) {
	type MultiFragsStruct struct {
		Role  string  `json:"role" frags:"description=User role in system,enum=admin|user|guest"`
		Email string  `json:"email" frags:"description=Contact email,format=email,pattern=^[a-z]+@[a-z]+\\.[a-z]+$"`
		Token *string `json:"token" frags:"x-session=auth-token,description=Auth token,title=Authentication Token"`
	}

	schema := StructToSchema(MultiFragsStruct{})

	assert.NotNil(t, schema)

	// Role - description and enum
	assert.Equal(t, "User role in system", schema.Properties["role"].Description)
	assert.Equal(t, []string{"admin", "user", "guest"}, schema.Properties["role"].Enum)

	// Email - description, format, and pattern
	assert.Equal(t, "Contact email", schema.Properties["email"].Description)
	assert.Equal(t, "email", schema.Properties["email"].Format)
	assert.Equal(t, "^[a-z]+@[a-z]+\\.[a-z]+$", schema.Properties["email"].Pattern)

	// Token - x-session, description, and title
	assert.NotNil(t, schema.Properties["token"].XSession)
	assert.Equal(t, "auth-token", *schema.Properties["token"].XSession)
	assert.Equal(t, "Auth token", schema.Properties["token"].Description)
	assert.Equal(t, "Authentication Token", schema.Properties["token"].Title)
}

// Test array types
func TestStructToSchema_ArrayTypes(t *testing.T) {
	type ArrayStruct struct {
		Tags    []string
		Numbers []int
		Flags   []bool
		Scores  []float64
	}

	schema := StructToSchema(ArrayStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, "array", schema.Properties["Tags"].Type)
	assert.Equal(t, "string", schema.Properties["Tags"].Items.Type)

	assert.Equal(t, "array", schema.Properties["Numbers"].Type)
	assert.Equal(t, "integer", schema.Properties["Numbers"].Items.Type)

	assert.Equal(t, "array", schema.Properties["Flags"].Type)
	assert.Equal(t, "boolean", schema.Properties["Flags"].Items.Type)

	assert.Equal(t, "array", schema.Properties["Scores"].Type)
	assert.Equal(t, "number", schema.Properties["Scores"].Items.Type)
}

// Test nested structs
func TestStructToSchema_NestedStructs(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
		Zip    string `json:"zip"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	schema := StructToSchema(Person{})

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)

	// Check nested address
	addressSchema := schema.Properties["address"]
	assert.NotNil(t, addressSchema)
	assert.Equal(t, "object", addressSchema.Type)
	assert.Len(t, addressSchema.Properties, 3)

	assert.Equal(t, "string", addressSchema.Properties["street"].Type)
	assert.Equal(t, "string", addressSchema.Properties["city"].Type)
	assert.Equal(t, "string", addressSchema.Properties["zip"].Type)

	assert.Contains(t, addressSchema.Required, "street")
	assert.Contains(t, addressSchema.Required, "city")
	assert.Contains(t, addressSchema.Required, "zip")
}

// Test array of structs
func TestStructToSchema_ArrayOfStructs(t *testing.T) {
	type Tag struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	type Document struct {
		Title string `json:"title"`
		Tags  []Tag  `json:"tags"`
	}

	schema := StructToSchema(Document{})

	assert.NotNil(t, schema)

	tagsSchema := schema.Properties["tags"]
	assert.Equal(t, "array", tagsSchema.Type)
	assert.NotNil(t, tagsSchema.Items)
	assert.Equal(t, "object", tagsSchema.Items.Type)

	assert.Len(t, tagsSchema.Items.Properties, 2)
	assert.Equal(t, "string", tagsSchema.Items.Properties["name"].Type)
	assert.Equal(t, "string", tagsSchema.Items.Properties["value"].Type)
}

// Test map types
func TestStructToSchema_MapTypes(t *testing.T) {
	type MapStruct struct {
		Metadata map[string]string
		Counters map[string]int
	}

	schema := StructToSchema(MapStruct{})

	assert.NotNil(t, schema)

	assert.Equal(t, "object", schema.Properties["Metadata"].Type)
	assert.Equal(t, "object", schema.Properties["Counters"].Type)
}

// Test unexported fields are ignored
func TestStructToSchema_UnexportedFields(t *testing.T) {
	type MixedStruct struct {
		Public  string
		private string
		Another string
	}

	schema := StructToSchema(MixedStruct{})

	assert.NotNil(t, schema)
	assert.Len(t, schema.Properties, 2)

	assert.Contains(t, schema.Properties, "Public")
	assert.Contains(t, schema.Properties, "Another")
	assert.NotContains(t, schema.Properties, "private")
}

// Test pointer to struct
func TestStructToSchema_PointerToStruct(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	// Test with pointer
	schema := StructToSchema(&TestStruct{})

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.Len(t, schema.Properties, 2)
	assert.Equal(t, "string", schema.Properties["Name"].Type)
	assert.Equal(t, "integer", schema.Properties["Age"].Type)
}

// Test complex real-world example
func TestStructToSchema_ComplexExample(t *testing.T) {
	type User struct {
		ID       int      `json:"id"`
		Username string   `json:"username" frags:"description=Unique username,pattern=^[a-zA-Z0-9_]+$"`
		Email    string   `json:"email" frags:"description=User email address,format=email"`
		Role     string   `json:"role" frags:"description=User role,enum=admin|moderator|user"`
		Active   bool     `json:"active" frags:"description=Account status"`
		Token    *string  `json:"token" frags:"x-session=user-session,description=Session token"`
		Tags     []string `json:"tags" frags:"description=User tags"`
		Score    *float64 `json:"score"`
	}

	schema := StructToSchema(User{})

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.Len(t, schema.Properties, 8)
	assert.Len(t, schema.Required, 8)

	// Verify ID
	assert.Equal(t, "integer", schema.Properties["id"].Type)

	// Verify Username with pattern
	assert.Equal(t, "string", schema.Properties["username"].Type)
	assert.Equal(t, "Unique username", schema.Properties["username"].Description)
	assert.Equal(t, "^[a-zA-Z0-9_]+$", schema.Properties["username"].Pattern)

	// Verify Email with format
	assert.Equal(t, "string", schema.Properties["email"].Type)
	assert.Equal(t, "email", schema.Properties["email"].Format)

	// Verify Role with enum
	assert.Equal(t, []string{"admin", "moderator", "user"}, schema.Properties["role"].Enum)

	// Verify Token with x-session and nullable
	assert.NotNil(t, schema.Properties["token"].XSession)
	assert.Equal(t, "user-session", *schema.Properties["token"].XSession)
	assert.NotNil(t, schema.Properties["token"].Nullable)
	assert.True(t, *schema.Properties["token"].Nullable)

	// Verify Tags array
	assert.Equal(t, "array", schema.Properties["tags"].Type)
	assert.Equal(t, "string", schema.Properties["tags"].Items.Type)

	// Verify nullable Score
	assert.NotNil(t, schema.Properties["score"].Nullable)
	assert.True(t, *schema.Properties["score"].Nullable)
	assert.Equal(t, "number", schema.Properties["score"].Type)
}

// Test nil input
func TestStructToSchema_NilInput(t *testing.T) {
	schema := StructToSchema(nil)
	assert.Nil(t, schema)
}

// Test non-struct input
func TestStructToSchema_NonStructInput(t *testing.T) {
	schema := StructToSchema("not a struct")
	assert.Nil(t, schema)

	schema = StructToSchema(123)
	assert.Nil(t, schema)

	schema = StructToSchema([]string{"a", "b"})
	assert.Nil(t, schema)
}
