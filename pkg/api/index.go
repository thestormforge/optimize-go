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
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	ParamOffset        = "offset"
	ParamLimit         = "limit"
	ParamLabelSelector = "labelSelector"
)

// IndexQuery represents the query parameter of an index resource.
type IndexQuery map[string][]string

// SetOffset sets the number of items to skip from the beginning of the index.
func (q IndexQuery) SetOffset(offset int) {
	if offset != 0 {
		url.Values(q).Set(ParamOffset, strconv.Itoa(offset))
	} else {
		url.Values(q).Del(ParamOffset)
	}
}

// SetLimit sets the maximum number of items to include with the index.
func (q IndexQuery) SetLimit(limit int) {
	if limit != 0 {
		url.Values(q).Set(ParamLimit, strconv.Itoa(limit))
	} else {
		url.Values(q).Del(ParamLimit)
	}
}

// SetLabelSelector is a helper to set label selectors used to filter the index.
func (q IndexQuery) SetLabelSelector(kv map[string]string) {
	ls := make([]string, 0, len(kv))
	for k, v := range kv {
		ls = append(ls, fmt.Sprintf("%s=%s", k, v))
	}
	if len(ls) > 0 {
		sort.Strings(ls)
		url.Values(q).Add(ParamLabelSelector, strings.Join(ls, ","))
	}
}
