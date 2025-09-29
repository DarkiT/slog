package dlp

import (
	"testing"
)

func TestTagParsingDebug(t *testing.T) {
	testCases := []struct {
		tag      string
		expected *DlpTagConfig
	}{
		{
			tag: "chinese_name",
			expected: &DlpTagConfig{
				Type:      "chinese_name",
				Recursive: false,
				Skip:      false,
				Custom:    "",
			},
		},
		{
			tag: ",recursive",
			expected: &DlpTagConfig{
				Type:      "",
				Recursive: true,
				Skip:      false,
				Custom:    "",
			},
		},
		{
			tag: "chinese_name,recursive",
			expected: &DlpTagConfig{
				Type:      "chinese_name",
				Recursive: true,
				Skip:      false,
				Custom:    "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {
			config, err := parseDlpTag(tc.tag)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			t.Logf("Tag: %s", tc.tag)
			t.Logf("Expected: %+v", tc.expected)
			t.Logf("Actual: %+v", config)

			if config == nil && tc.expected != nil {
				t.Error("Config should not be nil")
				return
			}

			if config != nil && tc.expected == nil {
				t.Error("Config should be nil")
				return
			}

			if config != nil && tc.expected != nil {
				if config.Type != tc.expected.Type {
					t.Errorf("Type mismatch: expected %s, got %s", tc.expected.Type, config.Type)
				}
				if config.Recursive != tc.expected.Recursive {
					t.Errorf("Recursive mismatch: expected %v, got %v", tc.expected.Recursive, config.Recursive)
				}
				if config.Skip != tc.expected.Skip {
					t.Errorf("Skip mismatch: expected %v, got %v", tc.expected.Skip, config.Skip)
				}
				if config.Custom != tc.expected.Custom {
					t.Errorf("Custom mismatch: expected %s, got %s", tc.expected.Custom, config.Custom)
				}
			}
		})
	}
}
