// Copyright (c) 2019 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package models

import (
	"errors"
	"fmt"
)

var validIDSchemes = []IDSchemeType{
	TypeLegacy,
	TypeQuoted,
	TypePrependMeta,
}

func (t IDSchemeType) validateIDSchemeType() error {
	if t == TypeDefault {
		return errors.New("id scheme type not set")
	}

	if t >= TypeLegacy && t <= TypePrependMeta {
		return nil
	}

	return fmt.Errorf("invalid config id schema type '%v': should be one of %v",
		t, validIDSchemes)
}

func (t IDSchemeType) String() string {
	switch t {
	case TypeDefault:
		return ""
	case TypeLegacy:
		return "legacy"
	case TypeQuoted:
		return "quoted"
	case TypePrependMeta:
		return "prependMeta"
	default:
		// Should never get here.
		return "unknown"
	}
}

// UnmarshalYAML unmarshals a stored merics type.
func (t *IDSchemeType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	for _, valid := range validIDSchemes {
		if str == valid.String() {
			*t = valid
			return nil
		}
	}

	return fmt.Errorf("invalid MetricsType '%s' valid types are: %v",
		str, validIDSchemes)
}