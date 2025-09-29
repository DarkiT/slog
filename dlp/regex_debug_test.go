package dlp

import (
	"regexp"
	"testing"
)

func TestChineseNameRegex(t *testing.T) {
	// 测试正则表达式本身
	t.Run("Regex pattern test", func(t *testing.T) {
		t.Logf("ChineseNamePattern: %s", ChineseNamePattern)

		names := []string{"张三", "李四", "王五", "赵六", "陈七"}

		for _, name := range names {
			matched := ChineseNameRegex.MatchString(name)
			t.Logf("Name: %s, Matched: %v", name, matched)
			if !matched {
				t.Errorf("Name %s should match", name)
			}
		}
	})

	// 测试匹配器
	t.Run("Matcher test", func(t *testing.T) {
		searcher := NewRegexSearcher()

		names := []string{"张三", "李四", "王五", "赵六", "陈七"}

		for _, name := range names {
			matches := searcher.SearchSensitiveByType(name, "chinese_name")
			t.Logf("Name: %s, Matches: %v", name, matches)
			if len(matches) == 0 {
				t.Errorf("Name %s should be found by matcher", name)
			}
		}
	})

	// 测试脱敏
	t.Run("Desensitization test", func(t *testing.T) {
		searcher := NewRegexSearcher()

		names := []string{"张三", "李四", "王五", "赵六", "陈七"}

		for _, name := range names {
			result := searcher.ReplaceParallel(name, "chinese_name")
			t.Logf("Name: %s, Result: %s", name, result)
			if result == name {
				t.Errorf("Name %s should be desensitized", name)
			}
		}
	})

	// 测试中文姓氏是否在列表中
	t.Run("Surname check", func(t *testing.T) {
		surnameRegex := regexp.MustCompile(ChineseSurnames)
		surnames := []string{"张", "李", "王", "赵", "陈"}

		for _, surname := range surnames {
			matched := surnameRegex.MatchString(surname)
			t.Logf("Surname: %s, Matched: %v", surname, matched)
			if !matched {
				t.Errorf("Surname %s should be in the list", surname)
			}
		}
	})
}
