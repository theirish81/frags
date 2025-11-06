package frags

import (
	"errors"
	"slices"
	"sort"
)

type Schema struct {
	AnyOf            []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Default          any                `json:"default,omitempty" yaml:"default,omitempty"`
	Description      string             `json:"description,omitempty" yaml:"description,omitempty"`
	Enum             []string           `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example          any                `json:"example,omitempty" yaml:"example,omitempty"`
	Format           string             `json:"format,omitempty" yaml:"format,omitempty"`
	Items            *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	MaxItems         *int64             `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MaxLength        *int64             `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MaxProperties    *int64             `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	Maximum          *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MinItems         *int64             `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MinLength        *int64             `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MinProperties    *int64             `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Minimum          *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Nullable         *bool              `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Pattern          string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Properties       map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	PropertyOrdering []string           `json:"propertyOrdering,omitempty" yaml:"propertyOrdering,omitempty"`
	Required         []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Title            string             `json:"title,omitempty" yaml:"title,omitempty"`
	Type             string             `json:"type,omitempty" yaml:"type,omitempty"`
	XPhase           *int               `json:"x-phase,omitempty" yaml:"x-phase,omitempty"`
}

func (s Schema) GetMaxPhase() int {
	p := -1
	for _, v := range s.Properties {
		if v.XPhase != nil {
			p = max(p, *v.XPhase)
		}
	}
	return p
}

func (s Schema) GetPhase(phase int) (Schema, error) {
	if s.GetMaxPhase() < phase {
		return s, errors.New("phase out of bound")
	}
	px := make(map[string]*Schema)
	req := make([]string, 0)
	for k, v := range s.Properties {
		if v.XPhase != nil && *v.XPhase == phase {
			px[k] = v
			if slices.Contains(s.Required, k) {
				req = append(req, k)
			}
		}
	}
	s.Properties = px
	s.Required = req
	return s, nil
}

func (s Schema) GetPhaseIndexes() []int {
	idx := make([]int, 0)
	for _, v := range s.Properties {
		if v.XPhase != nil {
			if !slices.Contains(idx, *v.XPhase) {
				idx = append(idx, *v.XPhase)
			}
		}
	}
	sort.Ints(idx)
	return idx
}
