// +build !ignore_autogenerated

/*
Copyright2019 The Kubernetes Authors.

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

// Code generated by helpgen. DO NOT EDIT.

package schemapatcher

import (
	"sigs.k8s.io/controller-tools/pkg/markers"
)

func (Generator) Help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "",
		DetailedHelp: markers.DetailedHelp{
			Summary: "patches existing CRDs with new schemata. ",
			Details: "It will generate output for each \"CRD Version\" (API version of the CRD type itself) , e.g. apiextensions/v1) available.",
		},
		FieldHelp: map[string]markers.DetailedHelp{
			"ManifestsPath": {
				Summary: "contains the CustomResourceDefinition YAML files.",
				Details: "",
			},
			"MaxDescLen": {
				Summary: "specifies the maximum description length for fields in CRD's OpenAPI schema. ",
				Details: "0 indicates drop the description for all fields completely. n indicates limit the description to at most n characters and truncate the description to closest sentence boundary if it exceeds n characters.",
			},
			"GenerateEmbeddedObjectMeta": {
				Summary: "specifies if any embedded ObjectMeta in the CRD should be generated",
				Details: "",
			},
		},
	}
}
