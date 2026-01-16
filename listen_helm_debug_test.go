package main

import (
	"strings"
	"testing"
)

func TestListenHelmDebug(t *testing.T) {
	tt := []struct {
		description string
		diff        string
		expect      string
		args        []string
	}{
		{"debug logs", absoluteTestPath("helm_debug_1.in"), absoluteTestPath("helm_debug_1.outt"), []string{}},
		{"no debug logs", absoluteTestPath("helm_debug_2.in"), absoluteTestPath("helm_debug_2.outt"), []string{}},
		{"drop helm manifests", absoluteTestPath("helm_debug_manifest.in"), absoluteTestPath("helm_debug_manifest_drop_manifests.outt"), []string{"--drop-helm-manifests"}},
		{"drop helm manifests helm prefix", absoluteTestPath("helm_debug_manifest_prefix.in"), absoluteTestPath("helm_debug_manifest_drop_manifests.outt"), []string{"--drop-helm-manifests"}},
		{"drop helm prefixed", absoluteTestPath("helm_debug_prefixed.in"), absoluteTestPath("helm_debug_prefixed_drop_prefixed.outt"), []string{"--drop-helm-prefixed"}},
	}
	for _, tc := range tt {
		t.Run(tc.description, func(t *testing.T) {
			input, expect := loadTestFiles(tc.diff, tc.expect)

			var output strings.Builder
			listenHelmDebug(input, &output, tc.args)
			assertEqual(t, expect, output)
		})
	}
}
