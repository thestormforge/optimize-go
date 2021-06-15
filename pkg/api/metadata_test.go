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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetadata_Headers(t *testing.T) {
	md := Metadata{}

	// Overwrite and ignore blanks
	md.SetTitle("Overwritten")
	md.SetTitle("Testing")
	assert.Equal(t, "Testing", md.Title())
	md.SetTitle("")
	assert.Equal(t, "Testing", md.Title())

	// Last-Modified ignores parse failures
	md["Last-Modified"] = []string{"fail"}
	assert.Equal(t, time.Time{}, md.LastModified())

	// We expect normalized header names
	http.Header(md).Set("location", "https://invalid.example.com/testing")
	assert.NotContains(t, md, "location")
	assert.Equal(t, "https://invalid.example.com/testing", md.Location())
}

func TestMetadata_Links(t *testing.T) {
	md := Metadata{}

	// Simple set/get
	md.SetLink(RelationNext, "https://invalid.example.com/list?offset=10")
	assert.Equal(t, "https://invalid.example.com/list?offset=10", md.Link(RelationNext))
	assert.Equal(t, []string{`<https://invalid.example.com/list?offset=10>;rel="next"`}, md["Link"])

	// Expand existing comma-delimited values on write
	md["Link"] = []string{`</foo>;rel=abc,</bar>;rel=xyz`}
	md.SetLink("abc", "/test")
	assert.Equal(t, []string{`</test>;rel="abc"`, `</bar>;rel="xyz"`}, md["Link"])

	// We do not normalize on set...
	delete(md, "Link")
	md.SetLink("previous", "/list?offset=0")
	assert.Equal(t, []string{`</list?offset=0>;rel="previous"`}, md["Link"])

	// ...but we do on get
	assert.Equal(t, "", md.Link("previous"))
	assert.Equal(t, "/list?offset=0", md.Link("prev"))
}
