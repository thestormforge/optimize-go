/*
Copyright 2021 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	RelationSelf        = "self"
	RelationNext        = "next"
	RelationPrev        = "prev"
	RelationAlternate   = "alternate"
	RelationUp          = "up"
	RelationLabels      = "https://stormforge.io/rel/labels"
	RelationTrials      = "https://stormforge.io/rel/trials"
	RelationNextTrial   = "https://stormforge.io/rel/next-trial"
	RelationScenarios   = "https://stormforge.io/rel/scenarios"
	RelationExperiments = "https://stormforge.io/rel/experiments"
	RelationTemplate    = "https://stormforge.io/rel/template"
)

// Metadata is used to hold single or multi-value metadata from list responses.
type Metadata map[string][]string

func (m Metadata) Title() string {
	return http.Header(m).Get("Title")
}

func (m Metadata) Location() string {
	return http.Header(m).Get("Location")
}

func (m Metadata) LastModified() time.Time {
	value, _ := http.ParseTime(http.Header(m).Get("Last-Modified"))
	return value
}

func (m Metadata) Link(rel string) string {
	for _, rh := range http.Header(m).Values("Link") {
		for _, h := range strings.Split(rh, ",") {
			r, l := splitLink(h)
			if strings.EqualFold(rel, r) {
				return l
			}
		}
	}
	return ""
}

func splitLink(value string) (rel, link string) {
	for _, l := range strings.Split(value, ";") {
		l = strings.Trim(l, " ")
		if l == "" {
			continue
		}

		if l[0] == '<' && l[len(l)-1] == '>' {
			link = strings.Trim(l, "<>")
			continue
		}

		p := strings.SplitN(l, "=", 2)
		if len(p) == 2 && strings.ToLower(p[0]) == "rel" {
			rel = strings.Trim(p[1], "\"")
			continue
		}
	}

	rel = CanonicalLinkRelation(rel)

	return
}

// CanonicalLinkRelation returns the supplied link relation name normalized for
// previously accepted values. The returned value can be compared case-insensitively
// to the supplied `Relation*` constants.
func CanonicalLinkRelation(rel string) string {
	switch strings.ToLower(rel) {
	case "previous":
		return RelationPrev

	case "https://carbonrelay.com/rel/labels",
		"https://carbonrelay.com/rel/triallabels":
		return RelationLabels

	case "https://carbonrelay.com/rel/trials":
		return RelationTrials

	case "https://carbonrelay.com/rel/next-trial",
		"https://carbonrelay.com/rel/nexttrial":
		return RelationNextTrial

	default:
		return rel
	}
}

// UnmarshalJSON extracts the supplied JSON, preserving the "_metadata" field if
// necessary. This should only be necessary on items in index (list) representations
// as top-level "_metadata" fields should normally be populated from HTTP headers.
func UnmarshalJSON(b []byte, v interface{}) error {
	if f := findMetadataField(reflect.ValueOf(v)); f.IsValid() {
		md := struct {
			Metadata jsonMetadata `json:"_metadata"`
		}{
			Metadata: jsonMetadata{},
		}
		if err := json.Unmarshal(b, &md); err == nil {
			f.Set(reflect.ValueOf(Metadata(md.Metadata)))
		}
	}

	return json.Unmarshal(b, v)
}

// findMetadataField searches for a `Metadata` typed field with a JSON tag of "-".
func findMetadataField(rv reflect.Value) reflect.Value {
	rv = reflect.Indirect(rv)
	for i := 0; i < rv.NumField(); i++ {
		ft := rv.Type().Field(i)
		if ft.Tag.Get("json") == "-" && ft.Type == reflect.TypeOf(Metadata{}) {
			return rv.Field(i)
		} else if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
			if result := findMetadataField(rv.Field(i)); result.IsValid() {
				return result
			}
		}
	}
	return reflect.Value{}
}

// jsonMetadata is a helper for unmarshalling a mapping where values may or
// may not be multi-valued strings. For example, `{"x":["y","z"],"a":"b"}` would
// wrap `"b"` in an array.
type jsonMetadata map[string][]string

func (m jsonMetadata) UnmarshalJSON(data []byte) error {
	// TODO Should `{"Link":"<x>;rel=x","Link":"<y>;rel=y"}` be allowed?
	md := make(map[string]interface{})
	if err := json.Unmarshal(data, &md); err != nil {
		return err
	}

	for k, v := range md {
		switch t := v.(type) {
		case string:
			m[k] = append(m[k], t)
		case []interface{}:
			for i := range t {
				m[k] = append(m[k], fmt.Sprintf("%s", t[i]))
			}
		}
	}

	return nil
}
