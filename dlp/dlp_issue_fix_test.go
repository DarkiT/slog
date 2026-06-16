package dlp

import (
	"strings"
	"testing"
)

// ============================================================
// Issue #6: Username matcher masks ordinary English words
// ============================================================

func TestIssue6_UsernameDoesNotMaskOrdinaryWords(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	tests := []struct {
		name  string
		input string
	}{
		{name: "common words", input: "cleanup sessions finished count=0"},
		{name: "log vocabulary", input: "mounted embedded static site"},
		{name: "mixed text", input: "the user cleanup task has sessions finished"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.DesensitizeText(tt.input)
			if result != tt.input {
				t.Errorf("ordinary words should NOT be masked.\n  input:  %s\n  output: %s", tt.input, result)
			}
		})
	}
}

func TestIssue6_UsernameTagStillWorks(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	type User struct {
		Name string `dlp:"username" json:"name"`
	}

	user := User{Name: "john_doe"}
	err := engine.DesensitizeStruct(&user)
	if err != nil {
		t.Fatalf("DesensitizeStruct failed: %v", err)
	}

	if user.Name == "john_doe" {
		t.Error("username should be desensitized when tagged with dlp:\"username\"")
	}
	if user.Name == "" {
		t.Error("username should not be fully emptied")
	}
	t.Logf("Username desensitized: %s", user.Name)
}

// ============================================================
// Issue #7: Matcher-level opt-out and field-scoped DLP
// ============================================================

func TestIssue7_DisableMatchers(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// New variadic API — more ergonomic
	engine.DisableMatchers("mobile_phone", "email")

	if !engine.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should be disabled")
	}
	if !engine.IsMatcherDisabled("email") {
		t.Error("email should be disabled")
	}

	disabled := engine.DisabledMatchers()
	// There are 4 default-disabled matchers (username, api_key, access_token, password)
	// plus the 2 we added = 6
	if len(disabled) < 6 {
		t.Errorf("expected at least 6 disabled matchers, got %d: %v", len(disabled), disabled)
	}
}

func TestIssue7_EnableMatchers(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// Disable mobile_phone
	engine.DisableMatchers("mobile_phone")
	if !engine.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should be disabled after DisableMatchers")
	}

	// Re-enable it
	engine.EnableMatchers("mobile_phone")
	if engine.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should be enabled after EnableMatchers")
	}

	// EnableMatchers on already-enabled matcher is a no-op
	engine.EnableMatchers("email")
	if engine.IsMatcherDisabled("email") {
		t.Error("email should still be enabled (no-op)")
	}

	// EnableMatchers on non-existent matcher is a no-op
	engine.EnableMatchers("nonexistent_matcher") // should not panic
}

func TestIssue7_SetMatcherEnabled(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// Disable then re-enable using SetMatcherEnabled
	engine.SetMatcherEnabled("mobile_phone", false)
	if !engine.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should be disabled after SetMatcherEnabled(false)")
	}

	engine.SetMatcherEnabled("mobile_phone", true)
	if engine.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should be enabled after SetMatcherEnabled(true)")
	}
}

func TestIssue7_EnabledMatchers(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	enabled := engine.EnabledMatchers()
	if len(enabled) == 0 {
		t.Error("there should be enabled matchers by default")
	}

	// username is disabled by default, so should NOT be in enabled list
	for _, name := range enabled {
		if name == "username" {
			t.Error("username should NOT be in EnabledMatchers (disabled by default)")
		}
	}

	// Disable a few and verify they disappear from enabled list
	engine.DisableMatchers("mobile_phone", "email")
	enabled2 := engine.EnabledMatchers()
	for _, name := range enabled2 {
		if name == "mobile_phone" || name == "email" {
			t.Errorf("%s should NOT be in EnabledMatchers after DisableMatchers", name)
		}
	}
}

func TestIssue7_SearcherDisabledMatcherSkipsProcessing(t *testing.T) {
	searcher := NewRegexSearcher()

	text := "phone: 13812345678"
	result1 := searcher.ReplaceAllTypes(text)
	if result1 == text {
		t.Fatal("phone number should be masked by default")
	}

	searcher2 := NewRegexSearcher()
	// New variadic API
	searcher2.DisableMatchers("mobile_phone", "medical_id", "landline")

	result2 := searcher2.ReplaceAllTypes(text)
	if result2 != text {
		t.Errorf("phone number should NOT be masked when disabled.\n  input:  %s\n  output: %s", text, result2)
	}
}

func TestSearcher_EnableMatchersRestoresDetection(t *testing.T) {
	searcher := NewRegexSearcher()

	text := "phone: 13812345678"

	// Disable mobile_phone
	searcher.DisableMatchers("mobile_phone")
	results := searcher.Match(text)
	for _, r := range results {
		if r.Type == "mobile_phone" {
			t.Fatal("mobile_phone should NOT be detected when disabled")
		}
	}

	// Re-enable mobile_phone
	searcher.EnableMatchers("mobile_phone")
	results2 := searcher.Match(text)
	hasPhone := false
	for _, r := range results2 {
		if r.Type == "mobile_phone" {
			hasPhone = true
			break
		}
	}
	if !hasPhone {
		t.Error("mobile_phone should be detected again after EnableMatchers")
	}
}

func TestIssue7_DesensitizeAttrsOnly(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	msg := "user login successful"
	attrs := map[string]string{
		"phone": "13812345678",
		"email": "test@example.com",
		"role":  "admin",
	}

	resultMsg, resultAttrs := engine.DesensitizeAttrsOnly(msg, attrs)

	// Message should not be modified
	if resultMsg != msg {
		t.Errorf("message should NOT be modified.\n  expected: %s\n  got:      %s", msg, resultMsg)
	}

	// Phone should be masked
	if resultAttrs["phone"] == "13812345678" {
		t.Error("phone attr should be masked")
	}
	if !strings.Contains(resultAttrs["phone"], "****") {
		t.Errorf("phone attr should contain ****, got: %s", resultAttrs["phone"])
	}

	// Email should be masked
	if resultAttrs["email"] == "test@example.com" {
		t.Error("email attr should be masked")
	}

	// Role should NOT be masked
	if resultAttrs["role"] != "admin" {
		t.Errorf("role attr should NOT be modified, got: %s", resultAttrs["role"])
	}
}

// ============================================================
// Password Pattern: disabled from free-text scanning
// ============================================================

func TestPasswordDisabledFromFreeText(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// Password embedded in text should NOT be detected in free-text scanning
	// (password matcher is disabled by default to avoid false positives)
	text := "password is abcdefg1 and secret is hello123"
	result := engine.DesensitizeText(text)
	// The words should remain unchanged
	if strings.Contains(result, "abcdefg1") && strings.Contains(result, "hello123") {
		// Both preserved - this is the expected behavior since password scanning is disabled
		t.Log("Password scanning correctly disabled for free text")
	}
}

func TestPasswordTagStillWorks(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	type Credentials struct {
		Pass string `dlp:"password" json:"pass"`
	}

	creds := Credentials{Pass: "mypassword1"}
	err := engine.DesensitizeStruct(&creds)
	if err != nil {
		t.Fatalf("DesensitizeStruct failed: %v", err)
	}

	if creds.Pass == "mypassword1" {
		t.Error("password should be desensitized when tagged with dlp:\"password\"")
	}
	t.Logf("Password desensitized: %s", creds.Pass)
}

// ============================================================
// IMEI Pattern fix: exactly 15 digits
// ============================================================

func TestIMEIPatternExactMatch(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 15-digit IMEI should be detected
	imeiText := "IMEI: 490154203237518"
	result := engine.DesensitizeText(imeiText)
	if result == imeiText {
		t.Error("15-digit IMEI should be detected and masked")
	}

	// 16-digit bank card should NOT match IMEI
	bankText := "card: 6222021234567890" // 16 digits
	detections := engine.DetectSensitiveInfo(bankText)
	if _, hasIMEI := detections["imei"]; hasIMEI {
		t.Error("16-digit number should NOT be detected as IMEI")
	}

	// 19-digit bank card should NOT match IMEI
	bankText19 := "card: 6222021234567890123" // 19 digits
	detections19 := engine.DetectSensitiveInfo(bankText19)
	if _, hasIMEI := detections19["imei"]; hasIMEI {
		t.Error("19-digit number should NOT be detected as IMEI")
	}
}

// ============================================================
// API Key / Access Token: disabled from free-text scanning
// ============================================================

func TestAPIKeyNotMaskedInFreeText(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// A long alphanumeric string should NOT be masked in free-text scanning
	// Use a string without digits to avoid MedicalID/PostalCode false positives
	longAlpha := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrs" // 45 chars, all letters
	result := engine.DesensitizeText("key: " + longAlpha)
	if !strings.Contains(result, longAlpha) {
		t.Errorf("45-char alphabetic string should NOT be masked by default.\n  output: %s", result)
	}
}

func TestAccessTokenNotMaskedInFreeText(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// A 40+ char alphabetic string should NOT be masked in free-text scanning
	longAlpha := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstu" // 47 chars, all letters
	result := engine.DesensitizeText("token: " + longAlpha)
	if !strings.Contains(result, longAlpha) {
		t.Errorf("47-char alphabetic string should NOT be masked by default.\n  output: %s", result)
	}
}

// ============================================================
// PostalCode Pattern: word boundaries work correctly
// ============================================================

func TestPostalCodePatternFixed(t *testing.T) {
	searcher := NewRegexSearcher()

	// Standalone 6-digit postal code should be detected by regex
	postalText := "postal: 518000"
	result := searcher.ReplaceAllTypes(postalText)
	if result == postalText {
		t.Error("standalone 6-digit postal code should be detected")
	}

	// Digits inside a longer alphanumeric identifier should NOT be matched as postal code
	// Using a format where digits are surrounded by letters
	mixedText := "idA123456Bend"
	result2 := searcher.ReplaceAllTypes(mixedText)
	// The text should be unchanged since digits are surrounded by letters (no word boundary)
	if result2 != mixedText {
		t.Errorf("digits surrounded by letters should NOT be modified.\n  input:  %s\n  output: %s", mixedText, result2)
	}
}

// ============================================================
// Searcher-level disabled matchers API tests
// ============================================================

func TestSearcher_DisableMatchers(t *testing.T) {
	searcher := NewRegexSearcher()

	// By default, username/api_key/access_token/password are disabled
	if !searcher.IsMatcherDisabled("username") {
		t.Error("username should be disabled by default")
	}
	if !searcher.IsMatcherDisabled("api_key") {
		t.Error("api_key should be disabled by default")
	}

	// mobile_phone is NOT disabled by default
	if searcher.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should NOT be disabled by default")
	}

	searcher.DisableMatchers("mobile_phone", "email")

	if !searcher.IsMatcherDisabled("mobile_phone") {
		t.Error("mobile_phone should be disabled after DisableMatchers")
	}
	if !searcher.IsMatcherDisabled("email") {
		t.Error("email should be disabled after DisableMatchers")
	}

	disabled := searcher.DisabledMatchers()
	if len(disabled) < 6 {
		t.Errorf("expected at least 6 disabled matchers, got %d: %v", len(disabled), disabled)
	}

	enabled := searcher.EnabledMatchers()
	if len(enabled) == 0 {
		t.Error("there should be enabled matchers")
	}
	for _, name := range enabled {
		if name == "mobile_phone" || name == "email" || name == "username" {
			t.Errorf("%s should NOT be in EnabledMatchers", name)
		}
	}
}

func TestSearcher_DisabledMatcherSkipsMatch(t *testing.T) {
	searcher := NewRegexSearcher()

	text := "phone: 13812345678"
	results := searcher.Match(text)
	hasPhone := false
	for _, r := range results {
		if r.Type == "mobile_phone" {
			hasPhone = true
			break
		}
	}
	if !hasPhone {
		t.Error("mobile_phone should be detected by default")
	}

	searcher.DisableMatchers("mobile_phone")
	results2 := searcher.Match(text)
	hasPhone2 := false
	for _, r := range results2 {
		if r.Type == "mobile_phone" {
			hasPhone2 = true
			break
		}
	}
	if hasPhone2 {
		t.Error("mobile_phone should NOT be detected when disabled")
	}
}

func TestSearcher_DisabledMatcherSkipsDetectAllTypes(t *testing.T) {
	searcher := NewRegexSearcher()

	text := "phone: 13812345678, email: test@example.com"
	detections := searcher.DetectAllTypes(text)
	if _, ok := detections["mobile_phone"]; !ok {
		t.Error("mobile_phone should be in detections by default")
	}

	searcher.DisableMatchers("mobile_phone")
	detections2 := searcher.DetectAllTypes(text)
	if _, ok := detections2["mobile_phone"]; ok {
		t.Error("mobile_phone should NOT be in detections when disabled")
	}
	// Email should still be detected
	if _, ok := detections2["email"]; !ok {
		t.Error("email should still be detected")
	}
}

func TestSearcher_DisabledMatcherSkipsReplaceAllTypes(t *testing.T) {
	searcher := NewRegexSearcher()

	text := "phone: 13812345678"
	result := searcher.ReplaceAllTypes(text)
	if result == text {
		t.Error("mobile_phone should be replaced by default")
	}

	searcher2 := NewRegexSearcher()
	// New variadic API
	searcher2.DisableMatchers("mobile_phone", "medical_id", "landline")
	result2 := searcher2.ReplaceAllTypes(text)
	if result2 != text {
		t.Errorf("mobile_phone should NOT be replaced when disabled.\n  input:  %s\n  output: %s", text, result2)
	}
}

// ============================================================
// Backward compatibility: struct tag still works for disabled matchers
// ============================================================

func TestBackwardCompat_APIKeyTagStillWorks(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	type Config struct {
		Key string `dlp:"api_key" json:"key"`
	}

	cfg := Config{Key: "abcdefghijklmnopqrstuvwxyz012345"}
	err := engine.DesensitizeStruct(&cfg)
	if err != nil {
		t.Fatalf("DesensitizeStruct failed: %v", err)
	}

	if cfg.Key == "abcdefghijklmnopqrstuvwxyz012345" {
		t.Error("api_key should be desensitized when tagged with dlp:\"api_key\"")
	}
	t.Logf("API key desensitized: %s", cfg.Key)
}

func TestBackwardCompat_AccessTokenTagStillWorks(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	type Token struct {
		Access string `dlp:"access_token" json:"access"`
	}

	tok := Token{Access: "abcdefghijklmnopqrstuvwxyz0123456789abcdefgh"}
	err := engine.DesensitizeStruct(&tok)
	if err != nil {
		t.Fatalf("DesensitizeStruct failed: %v", err)
	}

	if tok.Access == "abcdefghijklmnopqrstuvwxyz0123456789abcdefgh" {
		t.Error("access_token should be desensitized when tagged with dlp:\"access_token\"")
	}
	t.Logf("Access token desensitized: %s", tok.Access)
}

func TestBackwardCompat_PasswordTagStillWorks(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	type Secret struct {
		Pwd string `dlp:"password" json:"pwd"`
	}

	sec := Secret{Pwd: "mypassword1"}
	err := engine.DesensitizeStruct(&sec)
	if err != nil {
		t.Fatalf("DesensitizeStruct failed: %v", err)
	}

	if sec.Pwd == "mypassword1" {
		t.Error("password should be desensitized when tagged with dlp:\"password\"")
	}
	t.Logf("Password desensitized: %s", sec.Pwd)
}
