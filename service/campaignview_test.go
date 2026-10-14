package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/useragent"
)

// setNow fixes the service clock for the rest of the test.
func setNow(t *testing.T, now time.Time) {
	t.Helper()
	previous := nowUtc
	nowUtc = func() time.Time { return now }
	t.Cleanup(func() { nowUtc = previous })
}

func TestTrackRules(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)
	ctx := context.Background()
	views := f.svc.CampaignViews()

	c, err := f.svc.Campaigns().Create(admin, validCampaign("pixel"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := views.Track(ctx, TrackInput{CampaignId: c.Id, ExternalUserId: "", IpAddress: "10.0.0.1"}); err != nil {
		t.Fatalf("track without externalUserId: %v", err)
	}
	got, err := views.Track(ctx, TrackInput{CampaignId: c.Id, ExternalUserId: "user-1", IpAddress: "10.0.0.1"})
	if err != nil || got.Id != c.Id {
		t.Fatalf("track: campaign %v, err %v", got, err)
	}

	_, err = views.Track(ctx, TrackInput{CampaignId: "missing", ExternalUserId: "", IpAddress: "10.0.0.1"})
	assertErr(t, "unknown campaign", err, ErrNotFound)
	assertErr(t, "unknown campaign reason", err, ErrCampaignNotFound)

	_, err = views.Track(ctx, TrackInput{CampaignId: c.Id, ExternalUserId: strings.Repeat("x", 201), IpAddress: "10.0.0.1"})
	assertErr(t, "long externalUserId", err, ErrValidation)

	// Expired campaign.
	expired := *c
	expired.ExpireInUtc = time.Now().Add(-time.Minute)
	if err := f.repos.Campaigns().Update(ctx, f.tenantA.Id, &expired); err != nil {
		t.Fatal(err)
	}
	_, err = views.Track(ctx, TrackInput{CampaignId: c.Id, ExternalUserId: "", IpAddress: "10.0.0.1"})
	assertErr(t, "expired campaign", err, ErrNotFound)
	assertErr(t, "expired campaign reason", err, ErrCampaignExpired)
	if err := f.repos.Campaigns().Update(ctx, f.tenantA.Id, c); err != nil {
		t.Fatal(err)
	}

	// Inactive campaign.
	in := validCampaign("pixel")
	in.Active = false
	if _, err := f.svc.Campaigns().Update(admin, c.Id, in); err != nil {
		t.Fatal(err)
	}
	_, err = views.Track(ctx, TrackInput{CampaignId: c.Id, ExternalUserId: "", IpAddress: "10.0.0.1"})
	assertErr(t, "inactive campaign", err, ErrNotFound)
	assertErr(t, "inactive campaign reason", err, ErrCampaignInactive)
	if _, err := f.svc.Campaigns().Update(admin, c.Id, validCampaign("pixel")); err != nil {
		t.Fatal(err)
	}

	// Inactive tenant.
	if _, err := f.svc.Tenants().Update(f.asSuperAdmin(""), f.tenantA.Id, TenantInput{Name: "Tenant A", Active: false}); err != nil {
		t.Fatal(err)
	}
	_, err = views.Track(ctx, TrackInput{CampaignId: c.Id, ExternalUserId: "", IpAddress: "10.0.0.1"})
	assertErr(t, "inactive tenant", err, ErrNotFound)
	assertErr(t, "inactive tenant reason", err, ErrCampaignTenantInactive)

	// Views that were refused are not stored.
	summary, err := views.Summary(f.asSuperAdmin(f.tenantA.Id), c.Id, domain.Period{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalViews != 2 {
		t.Fatalf("total views = %d, want 2", summary.TotalViews)
	}
}

func TestViewReports(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)
	views := f.svc.CampaignViews()

	c, err := f.svc.Campaigns().Create(admin, validCampaign("report"))
	if err != nil {
		t.Fatal(err)
	}

	realNow := nowUtc
	day1 := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	day3 := day1.AddDate(0, 0, 2)
	track := func(at time.Time, externalUserId, ip string) {
		t.Helper()
		setNow(t, at)
		if _, err := views.Track(context.Background(), TrackInput{CampaignId: c.Id, ExternalUserId: externalUserId, IpAddress: ip}); err != nil {
			t.Fatal(err)
		}
	}
	track(day1.Add(9*time.Hour), "u1", "1.1.1.1")
	track(day1.Add(23*time.Hour+59*time.Minute), "u1", "2.2.2.2") // same user, other IP
	track(day1.Add(12*time.Hour), "u2", "1.1.1.1")
	track(day1.Add(13*time.Hour), "", "1.1.1.1") // anonymous: counted by IP
	track(day1.Add(14*time.Hour), "", "1.1.1.1") // same IP again
	track(day3.Add(8*time.Hour), "", "3.3.3.3")
	track(day3.Add(9*time.Hour), "3.3.3.3", "4.4.4.4") // external id equal to an IP is another visitor
	nowUtc = realNow                                   // the reads below use the real clock

	// Unique: u1, u2, ip 1.1.1.1, ip 3.3.3.3, external "3.3.3.3".
	summary, err := views.Summary(admin, c.Id, domain.Period{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalViews != 7 || summary.UniqueViews != 5 {
		t.Fatalf("summary = %+v, want total 7, unique 5", *summary)
	}

	// Period is [from, to).
	summary, err = views.Summary(admin, c.Id, domain.Period{FromUtc: day1.AddDate(0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalViews != 2 || summary.UniqueViews != 2 {
		t.Fatalf("summary from day2 = %+v, want 2/2", *summary)
	}
	summary, err = views.Summary(admin, c.Id, domain.Period{ToUtc: day1.Add(12 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalViews != 1 {
		t.Fatalf("summary until 12:00 = %+v, want 1 view", *summary)
	}

	days, err := views.Daily(admin, c.Id, domain.Period{})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.DailyCampaignViews{
		{DateUtc: day1, CampaignViewSummary: domain.CampaignViewSummary{TotalViews: 5, UniqueViews: 3}},
		{DateUtc: day1.AddDate(0, 0, 1)},
		{DateUtc: day3, CampaignViewSummary: domain.CampaignViewSummary{TotalViews: 2, UniqueViews: 2}},
	}
	if len(days) != len(want) {
		t.Fatalf("daily = %+v, want %+v", days, want)
	}
	for i := range want {
		if !days[i].DateUtc.Equal(want[i].DateUtc) || days[i].CampaignViewSummary != want[i].CampaignViewSummary {
			t.Errorf("day %d = %+v, want %+v", i, days[i], want[i])
		}
	}

	// An explicit period is filled from its first to its last day.
	days, err = views.Daily(admin, c.Id, domain.Period{FromUtc: day1.AddDate(0, 0, -1), ToUtc: day1.AddDate(0, 0, 5)})
	if err != nil || len(days) != 6 {
		t.Fatalf("daily with period: %d days, err %v; want 6", len(days), err)
	}

	_, err = views.Daily(admin, c.Id, domain.Period{FromUtc: day1, ToUtc: day1.AddDate(2, 0, 0)})
	assertErr(t, "daily too long", err, ErrValidation)
	_, err = views.Summary(admin, c.Id, domain.Period{FromUtc: day3, ToUtc: day1})
	assertErr(t, "inverted period", err, ErrValidation)

	list, err := views.List(admin, c.Id, domain.Period{}, Page{Size: 3})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 7 || len(list.Items) != 3 || !list.Items[0].ViewedOnUtc.Equal(day3.Add(9*time.Hour)) {
		t.Fatalf("list = total %d, %d items, first %v", list.Total, len(list.Items), list.Items[0].ViewedOnUtc)
	}
}

func TestTrackStoresSanitizedUserAgent(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)

	tests := []struct{ name, in, want string }{
		{"regular", "Mozilla/5.0 (Windows NT 10.0)", "Mozilla/5.0 (Windows NT 10.0)"},
		{"missing", "", ""},
		{"too long", strings.Repeat("á", 600), strings.Repeat("á", 512)},
		{"invalid utf-8 and NUL", "Bad\xff\x00Agent", "Bad�Agent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// One campaign per case, so its only view is the one tracked here.
			c, err := f.svc.Campaigns().Create(admin, validCampaign("ua "+tt.name))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.svc.CampaignViews().Track(context.Background(), TrackInput{
				CampaignId: c.Id, IpAddress: "10.0.0.1", UserAgent: tt.in,
			}); err != nil {
				t.Fatal(err)
			}
			list, err := f.svc.CampaignViews().List(admin, c.Id, domain.Period{}, Page{Size: 1})
			if err != nil {
				t.Fatal(err)
			}
			if got := list.Items[0].UserAgent; got != tt.want {
				t.Fatalf("user agent = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserAgentReport(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)
	views := f.svc.CampaignViews()
	c, err := f.svc.Campaigns().Create(admin, validCampaign("user agents"))
	if err != nil {
		t.Fatal(err)
	}

	const (
		gmail   = "Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)"
		iphone  = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
		mac     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)"
		chrome1 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
		chrome2 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
	)
	for _, v := range []struct{ user, ip, ua string }{
		{"u1", "1.1.1.1", gmail},
		{"u1", "1.1.1.1", gmail},
		{"u2", "1.1.1.1", gmail},
		{"u1", "2.2.2.2", iphone}, // u1 again, on the phone
		{"u3", "3.3.3.3", mac},
		{"u3", "3.3.3.3", chrome1},
		{"u3", "3.3.3.3", chrome2}, // same visitor, two Chrome versions: one unique Chrome view
		{"", "4.4.4.4", ""},
	} {
		if _, err := views.Track(context.Background(), TrackInput{CampaignId: c.Id, ExternalUserId: v.user, IpAddress: v.ip, UserAgent: v.ua}); err != nil {
			t.Fatal(err)
		}
	}

	report, err := views.UserAgents(admin, c.Id, domain.Period{})
	if err != nil {
		t.Fatal(err)
	}
	group := func(name string, total, unique int64) domain.ViewGroup {
		return domain.ViewGroup{Name: name, CampaignViewSummary: domain.CampaignViewSummary{TotalViews: total, UniqueViews: unique}}
	}
	check := func(dimension string, got, want []domain.ViewGroup) {
		t.Helper()
		if !slices.Equal(got, want) {
			t.Errorf("%s = %+v, want %+v", dimension, got, want)
		}
	}
	check("clients", report.Clients, []domain.ViewGroup{
		group("Gmail", 3, 2), group("Apple Mail", 2, 2), group("Chrome", 2, 1), group(useragent.Unknown, 1, 1),
	})
	check("operating systems", report.OperatingSystems, []domain.ViewGroup{
		group(useragent.Unknown, 4, 3), group("Windows", 2, 1), group("iOS", 1, 1), group("macOS", 1, 1),
	})
	check("devices", report.Devices, []domain.ViewGroup{
		group(useragent.Proxy, 3, 2), group(useragent.Desktop, 3, 1), group(useragent.Mobile, 1, 1), group(useragent.Unknown, 1, 1),
	})

	if len(report.UserAgents) != 6 {
		t.Fatalf("user agents = %+v, want 6", report.UserAgents)
	}
	first := report.UserAgents[0]
	if first.UserAgent != gmail || first.Client != "Gmail" || first.Device != useragent.Proxy || first.TotalViews != 3 || first.UniqueViews != 2 {
		t.Errorf("most viewed user agent = %+v", first)
	}
	// Ties are ordered by user agent, so the views without one come first.
	if none := report.UserAgents[1]; none.UserAgent != "" || none.Client != useragent.Unknown || none.TotalViews != 1 {
		t.Errorf("views without user agent = %+v", none)
	}

	_, err = views.UserAgents(f.as(domain.Admin, f.tenantB.Id), c.Id, domain.Period{})
	assertErr(t, "other tenant", err, ErrNotFound)
}

func TestEachView(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)
	views := f.svc.CampaignViews()
	c, err := f.svc.Campaigns().Create(admin, validCampaign("each view"))
	if err != nil {
		t.Fatal(err)
	}

	// More than one batch.
	start := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	for i := range eachViewBatchSize + 3 {
		setNow(t, start.Add(time.Duration(i)*time.Second))
		if _, err := views.Track(context.Background(), TrackInput{CampaignId: c.Id, IpAddress: "10.0.0.1"}); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}
	var previous time.Time
	err = views.EachView(admin, c.Id, domain.Period{}, func(v domain.CampaignView) error {
		if seen[v.Id] {
			t.Fatalf("view %s repeated", v.Id)
		}
		if !previous.IsZero() && v.ViewedOnUtc.After(previous) {
			t.Fatalf("views out of order: %v after %v", v.ViewedOnUtc, previous)
		}
		seen[v.Id], previous = true, v.ViewedOnUtc
		return nil
	})
	if err != nil || len(seen) != eachViewBatchSize+3 {
		t.Fatalf("each view: %d views, err %v; want %d", len(seen), err, eachViewBatchSize+3)
	}

	stop := errors.New("stop")
	calls := 0
	err = views.EachView(admin, c.Id, domain.Period{}, func(domain.CampaignView) error { calls++; return stop })
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("each view stopped after %d calls with %v", calls, err)
	}

	err = views.EachView(f.as(domain.Admin, f.tenantB.Id), c.Id, domain.Period{}, func(domain.CampaignView) error { return nil })
	assertErr(t, "other tenant", err, ErrNotFound)
}

func TestViewReportsAccess(t *testing.T) {
	f := newFixture(t)
	c, err := f.svc.Campaigns().Create(f.as(domain.Admin, f.tenantA.Id), validCampaign("access"))
	if err != nil {
		t.Fatal(err)
	}
	views := f.svc.CampaignViews()

	for _, role := range []domain.Role{domain.Admin, domain.ContentManager, domain.API} {
		if _, err := views.Summary(f.as(role, f.tenantA.Id), c.Id, domain.Period{}); err != nil {
			t.Errorf("%s: %v", role, err)
		}
	}

	_, err = views.Summary(f.as(domain.Admin, f.tenantB.Id), c.Id, domain.Period{})
	assertErr(t, "other tenant", err, ErrNotFound)
	_, err = views.Daily(f.as(domain.Admin, f.tenantB.Id), c.Id, domain.Period{})
	assertErr(t, "other tenant daily", err, ErrNotFound)
	_, err = views.List(f.as(domain.Admin, f.tenantB.Id), c.Id, domain.Period{}, Page{})
	assertErr(t, "other tenant list", err, ErrNotFound)

	_, err = views.Summary(f.asSuperAdmin(""), c.Id, domain.Period{})
	assertErr(t, "SuperAdmin without impersonation", err, ErrImpersonationRequired)

	imp, err := f.svc.Auth().Impersonate(f.asSuperAdmin(""), f.tenantA.Id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := views.Summary(auth.WithPrincipal(context.Background(), imp), c.Id, domain.Period{}); err != nil {
		t.Fatalf("SuperAdmin impersonating: %v", err)
	}

	_, err = views.Summary(context.Background(), c.Id, domain.Period{})
	assertErr(t, "unauthenticated", err, ErrUnauthenticated)
}
