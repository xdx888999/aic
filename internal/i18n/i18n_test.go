package i18n

import "testing"

func TestDetectLocaleFromLookup(t *testing.T) {
	lookup := func(key string) string {
		switch key {
		case "LANG":
			return "zh_CN.UTF-8"
		default:
			return ""
		}
	}

	actual := DetectLocaleFromLookup(lookup)
	if actual != LocaleChinese {
		t.Fatalf("期望检测到中文环境，实际为 %q", actual)
	}
}

func TestEnglishFallback(t *testing.T) {
	localizer := NewLocalizer(LocaleEnglish)
	actual := localizer.Text("scan_complete")
	if actual != "Scan complete" {
		t.Fatalf("期望英文文案为 %q，实际为 %q", "Scan complete", actual)
	}
}

func TestDetectLocaleRespectsAICLangOverride(t *testing.T) {
	lookup := func(key string) string {
		switch key {
		case "AIC_LANG":
			return "zh"
		case "LANG":
			return "en_US.UTF-8"
		default:
			return ""
		}
	}

	actual := DetectLocaleFromLookup(lookup)
	if actual != LocaleChinese {
		t.Fatalf("期望 AIC_LANG 优先覆盖系统语言，实际为 %q", actual)
	}
}

func TestDefaultLocaleIsEnglishWithoutOverride(t *testing.T) {
	if NewLocalizer(DefaultLocale()).Locale() != LocaleEnglish {
		t.Fatalf("期望默认启动语言为英文")
	}
}
