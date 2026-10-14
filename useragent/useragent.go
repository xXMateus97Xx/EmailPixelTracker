// Package useragent classifies the User-Agent of a pixel request into the
// e-mail client (or browser), operating system and device that opened the
// e-mail. It knows the e-mail specific cases that generic browser parsers get
// wrong: image proxies (Gmail, Yahoo), Apple Mail and the Outlook desktop app.
package useragent

import "strings"

// Names used by every dimension when it cannot be told.
const (
	// Unknown means the request had no User-Agent.
	Unknown = "Unknown"
	// Other means the User-Agent was not recognized.
	Other = "Other"
)

// Devices.
const (
	Desktop = "Desktop"
	Mobile  = "Mobile"
	Tablet  = "Tablet"
	// Proxy is an e-mail provider fetching the image for the recipient, which
	// hides the recipient's device and operating system.
	Proxy = "Email proxy"
	Bot   = "Bot"
)

type Info struct {
	// Client is the e-mail client or browser, e.g. "Gmail", "Apple Mail", "Chrome".
	Client string
	// OS is the operating system, e.g. "Windows", "iOS".
	OS     string
	Device string
}

type rule struct {
	tokens []string // any of them, case sensitive
	name   string
}

// proxies fetch the images on behalf of the recipient; their User-Agent
// describes the proxy, not the recipient (Gmail's claims to be Windows XP).
var proxies = []rule{
	{[]string{"GoogleImageProxy"}, "Gmail"},
	{[]string{"YahooMailProxy"}, "Yahoo Mail"},
}

// mailClients are checked before the browsers, since some of them embed a
// browser engine and its tokens.
var mailClients = []rule{
	{[]string{"Microsoft Outlook", "Outlook-iOS", "Outlook-Android", "ms-office"}, "Outlook"},
	{[]string{"Thunderbird/"}, "Thunderbird"},
	{[]string{"eM Client"}, "eM Client"},
}

// browsers are checked in order: Chromium based browsers also send "Chrome/",
// and every one of them sends "Safari/".
var browsers = []rule{
	{[]string{"Edg/", "EdgA/", "EdgiOS/", "Edge/"}, "Edge"},
	{[]string{"OPR/", "Opera"}, "Opera"},
	{[]string{"SamsungBrowser/"}, "Samsung Internet"},
	// An app showing the e-mail in a web view (e.g. the Samsung Email app).
	{[]string{"; wv)"}, "Android WebView"},
	{[]string{"Firefox/", "FxiOS/"}, "Firefox"},
	{[]string{"Chrome/", "CriOS/", "Chromium/"}, "Chrome"},
	{[]string{"Trident/", "MSIE "}, "Internet Explorer"},
}

var operatingSystems = []rule{
	{[]string{"Windows Phone"}, "Windows Phone"},
	{[]string{"Windows"}, "Windows"},
	{[]string{"iPad"}, "iPadOS"},
	{[]string{"iPhone", "iPod", "Outlook-iOS"}, "iOS"},
	{[]string{"Android"}, "Android"},
	{[]string{"CrOS"}, "ChromeOS"},
	// After iOS, whose User-Agent says "like Mac OS X".
	{[]string{"Macintosh", "Mac OS X"}, "macOS"},
	{[]string{"Linux", "X11"}, "Linux"},
}

var botTokens = []string{"bot", "crawl", "spider", "headless", "curl/", "wget/", "python", "go-http-client", "java/", "postman", "preview"}

func match(ua string, rules []rule) string {
	for _, r := range rules {
		for _, token := range r.tokens {
			if strings.Contains(ua, token) {
				return r.name
			}
		}
	}
	return ""
}

func orOther(name string) string {
	if name == "" {
		return Other
	}
	return name
}

func Parse(ua string) Info {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return Info{Client: Unknown, OS: Unknown, Device: Unknown}
	}

	if proxy := match(ua, proxies); proxy != "" {
		return Info{Client: proxy, OS: Unknown, Device: Proxy}
	}

	os := orOther(match(ua, operatingSystems))
	lower := strings.ToLower(ua)
	for _, token := range botTokens {
		if strings.Contains(lower, token) {
			return Info{Client: Bot, OS: os, Device: Bot}
		}
	}

	client := match(ua, mailClients)
	if client == "" {
		client = match(ua, browsers)
	}
	if client == "" && strings.Contains(ua, "AppleWebKit/") && (os == "macOS" || os == "iOS" || os == "iPadOS") {
		// Safari sends "Version/x Safari/y"; Apple Mail (and other apps
		// showing the e-mail in a WebKit view) send neither.
		if strings.Contains(ua, "Safari/") {
			client = "Safari"
		} else {
			client = "Apple Mail"
		}
	}
	if client == "" && strings.Contains(ua, "Safari/") && strings.Contains(ua, "Version/") {
		client = "Safari"
	}

	return Info{Client: orOther(client), OS: os, Device: device(ua, os)}
}

func device(ua, os string) string {
	switch {
	// Android browsers send "Mobile" only on phones; apps such as
	// "Outlook-Android/2.0" send no browser tokens at all.
	case os == "iPadOS" || strings.Contains(ua, "Tablet") ||
		(os == "Android" && strings.Contains(ua, "Mozilla/") && !strings.Contains(ua, "Mobile")):
		return Tablet
	case os == "iOS" || os == "Android" || os == "Windows Phone" || strings.Contains(ua, "Mobi"):
		return Mobile
	case os == "Windows" || os == "macOS" || os == "Linux" || os == "ChromeOS":
		return Desktop
	}
	return Other
}
