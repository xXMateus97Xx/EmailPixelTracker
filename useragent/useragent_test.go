package useragent

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		ua   string
		want Info
	}{
		{"", Info{Unknown, Unknown, Unknown}},
		{"   ", Info{Unknown, Unknown, Unknown}},
		{"Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)",
			Info{"Gmail", Unknown, Proxy}},
		{"YahooMailProxy; https://help.yahoo.com/kb/yahoo-mail-proxy-SLN28749.html",
			Info{"Yahoo Mail", Unknown, Proxy}},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)",
			Info{"Apple Mail", "macOS", Desktop}},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
			Info{"Apple Mail", "iOS", Mobile}},
		{"Mozilla/5.0 (iPad; CPU OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
			Info{"Apple Mail", "iPadOS", Tablet}},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			Info{"Safari", "iOS", Mobile}},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
			Info{"Safari", "macOS", Desktop}},
		{"Microsoft Office/16.0 (Windows NT 10.0; Microsoft Outlook 16.0.17126; Pro)",
			Info{"Outlook", "Windows", Desktop}},
		{"Outlook-iOS/2.0", Info{"Outlook", "iOS", Mobile}},
		{"Outlook-Android/2.0", Info{"Outlook", "Android", Mobile}},
		{"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Thunderbird/128.3.0",
			Info{"Thunderbird", "Linux", Desktop}},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			Info{"Chrome", "Windows", Desktop}},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
			Info{"Edge", "Windows", Desktop}},
		{"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
			Info{"Chrome", "Android", Mobile}},
		{"Mozilla/5.0 (Linux; Android 13; SM-X700) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			Info{"Chrome", "Android", Tablet}},
		{"Mozilla/5.0 (Linux; Android 14; SM-S918B; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/129.0.0.0 Mobile Safari/537.36",
			Info{"Android WebView", "Android", Mobile}},
		{"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/26.0 Chrome/122.0.0.0 Mobile Safari/537.36",
			Info{"Samsung Internet", "Android", Mobile}},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.6; rv:131.0) Gecko/20100101 Firefox/131.0",
			Info{"Firefox", "macOS", Desktop}},
		{"Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			Info{"Chrome", "ChromeOS", Desktop}},
		{"curl/8.9.1", Info{Bot, Other, Bot}},
		{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", Info{Bot, Other, Bot}},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/129.0.0.0 Safari/537.36",
			Info{Bot, "Linux", Bot}},
		{"TestMailClient/1.0", Info{Other, Other, Other}},
	}
	for _, tt := range tests {
		if got := Parse(tt.ua); got != tt.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.ua, got, tt.want)
		}
	}
}
