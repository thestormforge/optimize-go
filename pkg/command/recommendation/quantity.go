/*
Copyright 2022 GramLabs, Inc.

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

package recommendation

import (
	"github.com/thestormforge/optimize-go/pkg/api"
)

// QuantityLess returns true if 'a' is strictly less than 'b'.
func QuantityLess(a, b *api.NumberOrString) bool {
	af, bf := a.Quantity(), b.Quantity()

	if af == nil {
		return bf != nil
	} else if bf == nil {
		return af != nil
	}

	return af.Cmp(bf) < 0
}
