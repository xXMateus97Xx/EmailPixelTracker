package api

import (
	"bytes"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/xuri/excelize/v2"
)

func (c *client) raw(path string) *httptest.ResponseRecorder {
	c.t.Helper()
	return c.rawAs(path, "TestMailClient/1.0", "203.0.113.7")
}

// rawAs requests path without a token, from the given user agent and IP.
func (c *client) rawAs(path, userAgent, ip string) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = ip + ":51000"
	req.Header.Set("User-Agent", userAgent)
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	return rec
}

// download GETs a file; only the status and headers are checked against openapi.yaml.
func (c *client) download(path, token string, wantStatus int) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		c.t.Fatalf("GET %s: status %d, want %d; body: %s", path, rec.Code, wantStatus, rec.Body.String())
	}
	route, pathParams, err := c.spec.FindRoute(req)
	if err != nil {
		c.t.Fatalf("GET %s: not found in openapi.yaml: %v", path, err)
	}
	err = openapi3filter.ValidateResponse(req.Context(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: pathParams, Route: route},
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, ExcludeResponseBody: rec.Code == http.StatusOK},
	})
	if err != nil {
		c.t.Fatalf("GET %s: response %d does not match openapi.yaml: %v", path, rec.Code, err)
	}
	return rec
}

func TestUserAgentReportsAndExport(t *testing.T) {
	c := newClient(t)
	root := c.login("root", "rootpassword")
	var tenant tenantResponse
	c.do(http.MethodPost, "/api/tenants", root, map[string]any{"name": "Acme"}, http.StatusCreated, &tenant)
	var imp tokenResponse
	c.do(http.MethodPost, "/api/auth/impersonate", root, impersonateRequest{TenantId: tenant.Id}, http.StatusOK, &imp)
	var campaign campaignResponse
	c.do(http.MethodPost, "/api/campaigns", imp.Token, map[string]any{
		"name": "Black Friday 2026!", "expireInUtc": time.Now().Add(48 * time.Hour), "pixelColorA": 1,
	}, http.StatusCreated, &campaign)

	const (
		gmail  = "Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)"
		iphone = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
	)
	c.rawAs("/pixel/"+campaign.Id+"?externalUserId=ana", gmail, "198.51.100.1")
	c.rawAs("/pixel/"+campaign.Id+"?externalUserId=ana", gmail, "198.51.100.1")
	c.rawAs("/pixel/"+campaign.Id+"?externalUserId=bia", iphone, "198.51.100.2")
	c.rawAs("/pixel/"+campaign.Id, "", "198.51.100.3")

	reports := "/api/campaigns/" + campaign.Id + "/reports/"
	var report userAgentReportResponse
	c.do(http.MethodGet, reports+"user-agents", imp.Token, nil, http.StatusOK, &report)
	if len(report.Clients) != 3 || report.Clients[0] != (viewGroupResponse{Name: "Gmail", TotalViews: 2, UniqueViews: 1}) {
		t.Errorf("clients %+v", report.Clients)
	}
	if len(report.Devices) != 3 || report.Devices[0].Name != "Email proxy" {
		t.Errorf("devices %+v", report.Devices)
	}
	if len(report.OperatingSystems) != 2 || len(report.UserAgents) != 3 || report.UserAgents[0].UserAgent != gmail {
		t.Errorf("report %+v", report)
	}
	c.do(http.MethodGet, reports+"user-agents?toUtc=bad", imp.Token, nil, http.StatusBadRequest, nil)
	c.do(http.MethodGet, "/api/campaigns/unknown/reports/user-agents", imp.Token, nil, http.StatusNotFound, nil)

	rec := c.download(reports+"export?fromUtc=2026-01-01T00:00:00Z", imp.Token, http.StatusOK)
	if ct := rec.Header().Get("Content-Type"); ct != xlsxContentType {
		t.Fatalf("content type %q", ct)
	}
	wantName := "relatorio-black-friday-2026-20260101-" + time.Now().UTC().Format("20060102") + ".xlsx"
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename=`+wantName {
		t.Errorf("content disposition %q, want file %s", cd, wantName)
	}

	f, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, want := f.GetSheetList(), []string{"Resumo", "Por dia", "Clientes de e-mail", "Sistemas operacionais", "Dispositivos", "User agents", "Visualizações"}; !slices.Equal(got, want) {
		t.Fatalf("sheets %v, want %v", got, want)
	}
	rows := func(sheet string) [][]string {
		t.Helper()
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	if summary := rows("Resumo"); summary[0][1] != "Black Friday 2026!" || summary[4][1] != "4" || summary[5][1] != "3" {
		t.Errorf("summary sheet %v", summary)
	}
	if daily := rows("Por dia"); len(daily) < 2 || daily[len(daily)-1][1] != "4" {
		t.Errorf("daily sheet %v", daily)
	}
	if clients := rows("Clientes de e-mail"); len(clients) != 4 || clients[1][0] != "Gmail" || clients[1][3] != "50.00%" {
		t.Errorf("clients sheet %v", clients)
	}
	if devices := rows("Dispositivos"); devices[1][0] != "Proxy de e-mail" {
		t.Errorf("devices sheet %v", devices)
	}
	views := rows("Visualizações")
	if len(views) != 5 || views[0][0] != "Quando (UTC)" {
		t.Fatalf("views sheet %v", views)
	}
	// Views tracked in the same clock tick have no defined order.
	if !slices.ContainsFunc(views[1:], func(row []string) bool {
		return len(row) == 7 && row[1] == "ana" && row[2] == "198.51.100.1" && row[4] == "Gmail" && row[6] == "Proxy de e-mail"
	}) {
		t.Errorf("views sheet has no Gmail view of ana: %v", views)
	}

	// Views beyond the rows of a sheet are left out, and the summary says so.
	previous := maxSheetViews
	maxSheetViews = 2
	t.Cleanup(func() { maxSheetViews = previous })
	f2, err := excelize.OpenReader(bytes.NewReader(c.download(reports+"export", imp.Token, http.StatusOK).Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	if views, _ := f2.GetRows("Visualizações"); len(views) != 3 {
		t.Errorf("cut views sheet has %d rows, want 3", len(views))
	}
	if summary, _ := f2.GetRows("Resumo"); !strings.Contains(summary[len(summary)-1][0], "2 mais recentes de 4") {
		t.Errorf("summary does not tell the views were cut: %v", summary)
	}

	c.download(reports+"export?fromUtc=bad", imp.Token, http.StatusBadRequest)
	c.download("/api/campaigns/unknown/reports/export", imp.Token, http.StatusNotFound)
	c.download(reports+"export", "", http.StatusUnauthorized)
}

func TestPixelAndReports(t *testing.T) {
	c := newClient(t)
	root := c.login("root", "rootpassword")

	var tenant tenantResponse
	c.do(http.MethodPost, "/api/tenants", root, map[string]any{"name": "Acme"}, http.StatusCreated, &tenant)
	var imp tokenResponse
	c.do(http.MethodPost, "/api/auth/impersonate", root, impersonateRequest{TenantId: tenant.Id}, http.StatusOK, &imp)

	var campaign campaignResponse
	c.do(http.MethodPost, "/api/campaigns", imp.Token, map[string]any{
		"name": "Newsletter", "expireInUtc": time.Now().Add(48 * time.Hour),
		"pixelColorR": 255, "pixelColorG": 0, "pixelColorB": 128, "pixelColorA": 0.5,
	}, http.StatusCreated, &campaign)

	// The pixel is a 1x1 PNG with the campaign color.
	rec := c.raw("/pixel/" + campaign.Id + "?externalUserId=customer-42")
	if rec.Code != http.StatusOK {
		t.Fatalf("pixel status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content type %q", ct)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("pixel must not be cached, Cache-Control: %q", rec.Header().Get("Cache-Control"))
	}
	first := rec.Body.Bytes()
	img, err := png.Decode(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 1 || b.Dy() != 1 {
		t.Fatalf("image size %v", b)
	}
	if got, want := color.NRGBAModel.Convert(img.At(0, 0)), (color.NRGBA{R: 255, G: 0, B: 128, A: 128}); got != want {
		t.Fatalf("pixel color %v, want %v", got, want)
	}

	// Same color is served from the cache; anonymous views count by IP.
	second := c.raw("/pixel/" + campaign.Id).Body.Bytes()
	if !bytes.Equal(first, second) {
		t.Fatal("cached pixel differs")
	}
	c.raw("/pixel/" + campaign.Id)

	if rec := c.raw("/pixel/unknown"); rec.Code != http.StatusNotFound || rec.Body.Len() != 0 {
		t.Fatalf("unknown campaign: status %d, body %q", rec.Code, rec.Body.String())
	}

	var summary viewSummaryResponse
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/summary", imp.Token, nil, http.StatusOK, &summary)
	if summary.TotalViews != 3 || summary.UniqueViews != 2 {
		t.Fatalf("summary %+v, want total 3, unique 2", summary)
	}

	var daily dailyViewsResponse
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/daily", imp.Token, nil, http.StatusOK, &daily)
	today := time.Now().UTC().Format(time.DateOnly)
	if len(daily.Days) != 1 || daily.Days[0].Date != today || daily.Days[0].TotalViews != 3 {
		t.Fatalf("daily %+v", daily)
	}

	var views pageResponse[campaignViewResponse]
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/views?pageSize=2", imp.Token, nil, http.StatusOK, &views)
	if views.Total != 3 || len(views.Items) != 2 || views.Items[0].IpAddress != "203.0.113.7" ||
		views.Items[0].UserAgent != "TestMailClient/1.0" {
		t.Fatalf("views %+v", views)
	}

	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/summary?fromUtc="+future, imp.Token, nil, http.StatusOK, &summary)
	if summary.TotalViews != 0 {
		t.Fatalf("summary from the future %+v", summary)
	}
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/summary?fromUtc=yesterday", imp.Token, nil, http.StatusBadRequest, nil)
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/summary", "", nil, http.StatusUnauthorized, nil)

	// API users of the tenant can read the reports.
	c.do(http.MethodPost, "/api/users", imp.Token, map[string]any{
		"name": "acme-api", "username": "acme-api", "password": "apipassword", "role": "API",
	}, http.StatusCreated, nil)
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/summary", c.login("acme-api", "apipassword"), nil, http.StatusOK, nil)

	// A deactivated campaign answers 404 and is no longer counted.
	c.do(http.MethodPut, "/api/campaigns/"+campaign.Id, imp.Token, map[string]any{
		"name": "Newsletter", "active": false, "expireInUtc": time.Now().Add(48 * time.Hour), "pixelColorA": 1,
	}, http.StatusOK, nil)
	if rec := c.raw("/pixel/" + campaign.Id); rec.Code != http.StatusNotFound {
		t.Fatalf("inactive campaign: status %d", rec.Code)
	}
	c.do(http.MethodGet, "/api/campaigns/"+campaign.Id+"/reports/summary", imp.Token, nil, http.StatusOK, &summary)
	if summary.TotalViews != 3 {
		t.Fatalf("inactive campaign was counted: %+v", summary)
	}
}
