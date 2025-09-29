package dlp

import (
	"testing"
)

func TestBankCardDesensitization(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 测试银行卡脱敏
	t.Run("Bank card test", func(t *testing.T) {
		bankCard := "6222020000000000000"
		result := engine.DesensitizeSpecificType(bankCard, "bank_card")
		t.Logf("Original: %s, Desensitized: %s", bankCard, result)

		if result == bankCard {
			t.Error("Bank card should be desensitized")
		}
	})

	// 测试结构体中的银行卡脱敏
	t.Run("Struct bank card test", func(t *testing.T) {
		type User struct {
			BankCard string `dlp:"bank_card"`
		}

		user := &User{BankCard: "6222020000000000000"}
		t.Logf("Before: %+v", user)

		err := engine.DesensitizeStructAdvanced(user)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", user)
		if user.BankCard == "6222020000000000000" {
			t.Error("Bank card should be desensitized")
		}
	})

	// 测试不同的银行卡号
	t.Run("Different bank cards", func(t *testing.T) {
		cards := []string{
			"6222020000000000000",
			"4111111111111111",
			"5555555555554444",
		}

		for _, card := range cards {
			result := engine.DesensitizeSpecificType(card, "bank_card")
			t.Logf("Card: %s, Result: %s", card, result)
		}
	})
}
