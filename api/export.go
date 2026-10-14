package api

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/useragent"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// maxSheetViews is how many views fit in a sheet: Excel's 1,048,576 rows
// minus the header.
var maxSheetViews int64 = 1_048_575

var errSheetFull = errors.New("sheet full")

// Names of the useragent package shown in the spreadsheet. Product names
// (Gmail, Windows...) are kept as they are.
var userAgentLabels = map[string]string{
	useragent.Unknown: "Desconhecido",
	useragent.Other:   "Outro",
	useragent.Desktop: "Computador",
	useragent.Mobile:  "Celular",
	useragent.Tablet:  "Tablet",
	useragent.Proxy:   "Proxy de e-mail",
	useragent.Bot:     "Robô / automação",
}

func userAgentLabel(name string) string {
	if label, ok := userAgentLabels[name]; ok {
		return label
	}
	return name
}

// exportCampaignReports answers every report of the campaign (summary, daily,
// user agents and every view) as an Excel workbook, one sheet per report.
func (h *handler) exportCampaignReports(c *gin.Context) {
	period, ok := periodFromQuery(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	views := h.svc.CampaignViews()

	campaign, err := h.svc.Campaigns().GetById(ctx, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}

	// Every sheet counts the same views: the ones tracked until now. This
	// also keeps the views that arrive during the export from shifting the
	// batches of the views sheet.
	now := time.Now().UTC()
	// The period ends after the current millisecond, so a view tracked in
	// the same clock tick is counted (MongoDB keeps milliseconds).
	end := now.Truncate(time.Millisecond).Add(time.Millisecond)
	if (period.ToUtc.IsZero() || period.ToUtc.After(end)) && period.FromUtc.Before(end) {
		period.ToUtc = end
	}

	summary, err := views.Summary(ctx, campaign.Id, period)
	if err != nil {
		writeError(c, err)
		return
	}
	daily, err := views.Daily(ctx, campaign.Id, period)
	if err != nil {
		writeError(c, err)
		return
	}
	userAgents, err := views.UserAgents(ctx, campaign.Id, period)
	if err != nil {
		writeError(c, err)
		return
	}

	w, err := newWorkbook()
	if err != nil {
		writeError(c, err)
		return
	}
	defer w.close()

	if err := w.writeReports(campaign, period, summary, daily, userAgents, now, func(fn func(domain.CampaignView) error) error {
		return views.EachView(ctx, campaign.Id, period, fn)
	}); err != nil {
		writeError(c, err)
		return
	}

	c.Header("Content-Type", xlsxContentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": exportFileName(campaign.Name, period),
	}))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	// The status is already sent; a failure here is a broken connection.
	_ = w.file.Write(c.Writer)
}

// exportFileName is "relatorio-<campaign>-<from>-<to>.xlsx", with the
// campaign name reduced to letters, digits and dashes.
func exportFileName(campaign string, period domain.Period) string {
	slug := strings.Join(strings.FieldsFunc(strings.ToLower(campaign), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), "-")
	if slug == "" {
		slug = "campanha"
	}
	name := "relatorio-" + slug
	if !period.FromUtc.IsZero() {
		name += "-" + period.FromUtc.Format("20060102")
	}
	if !period.ToUtc.IsZero() {
		name += "-" + period.ToUtc.Format("20060102")
	}
	return name + ".xlsx"
}

type workbook struct {
	file *excelize.File
	// Style ids.
	header, dateTime, date, percent, wrap int
}

func newWorkbook() (*workbook, error) {
	f := excelize.NewFile()
	w := &workbook{file: f}
	styles := []struct {
		dst   *int
		style *excelize.Style
	}{
		{&w.header, &excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E4E4E7"}},
		}},
		{&w.dateTime, &excelize.Style{CustomNumFmt: new("dd/mm/yyyy hh:mm:ss")}},
		{&w.date, &excelize.Style{CustomNumFmt: new("dd/mm/yyyy")}},
		{&w.percent, &excelize.Style{NumFmt: 10}}, // 0.00%
		{&w.wrap, &excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}}},
	}
	for _, s := range styles {
		id, err := f.NewStyle(s.style)
		if err != nil {
			f.Close()
			return nil, err
		}
		*s.dst = id
	}
	return w, nil
}

func (w *workbook) close() { _ = w.file.Close() }

// sheet writes the rows of one sheet, in order.
type sheet struct {
	sw   *excelize.StreamWriter
	rows int
}

// Sheets of the workbook, in order.
const (
	summarySheet    = "Resumo"
	dailySheet      = "Por dia"
	clientsSheet    = "Clientes de e-mail"
	systemsSheet    = "Sistemas operacionais"
	devicesSheet    = "Dispositivos"
	userAgentsSheet = "User agents"
	viewsSheet      = "Visualizações"
)

// addSheets creates the sheets in order; the first one replaces the default "Sheet1".
func (w *workbook) addSheets(names ...string) error {
	for i, name := range names {
		if i == 0 {
			if err := w.file.SetSheetName(w.file.GetSheetName(0), name); err != nil {
				return err
			}
			continue
		}
		if _, err := w.file.NewSheet(name); err != nil {
			return err
		}
	}
	return nil
}

// stream starts writing an existing sheet with the given column widths; a
// header row, if given, stays frozen.
func (w *workbook) stream(name string, widths []float64, header ...string) (*sheet, error) {
	sw, err := w.file.NewStreamWriter(name)
	if err != nil {
		return nil, err
	}
	for i, width := range widths {
		if err := sw.SetColWidth(i+1, i+1, width); err != nil {
			return nil, err
		}
	}
	s := &sheet{sw: sw}
	if len(header) > 0 {
		if err := sw.SetPanes(&excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
			return nil, err
		}
		cells := make([]any, len(header))
		for i, h := range header {
			cells[i] = excelize.Cell{StyleID: w.header, Value: h}
		}
		if err := s.add(cells...); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *sheet) add(values ...any) error {
	s.rows++
	cell, err := excelize.CoordinatesToCellName(1, s.rows)
	if err != nil {
		return err
	}
	return s.sw.SetRow(cell, values)
}

func (s *sheet) flush() error { return s.sw.Flush() }

func (w *workbook) writeReports(
	campaign *domain.Campaign,
	period domain.Period,
	summary *domain.CampaignViewSummary,
	daily []domain.DailyCampaignViews,
	userAgents *domain.UserAgentReport,
	generatedAt time.Time,
	eachView func(func(domain.CampaignView) error) error,
) error {
	err := w.addSheets(summarySheet, dailySheet, clientsSheet, systemsSheet, devicesSheet, userAgentsSheet, viewsSheet)
	if err != nil {
		return err
	}
	// The views sheet is written first, since the summary tells whether it had to be cut.
	listed, err := w.writeViews(eachView)
	if err != nil {
		return err
	}
	if err := w.writeSummary(campaign, period, summary, listed, generatedAt); err != nil {
		return err
	}
	if err := w.writeDaily(daily); err != nil {
		return err
	}
	for _, d := range []struct {
		sheet, column string
		groups        []domain.ViewGroup
	}{
		{clientsSheet, "Cliente de e-mail / navegador", userAgents.Clients},
		{systemsSheet, "Sistema operacional", userAgents.OperatingSystems},
		{devicesSheet, "Dispositivo", userAgents.Devices},
	} {
		if err := w.writeGroups(d.sheet, d.column, d.groups, summary.TotalViews); err != nil {
			return err
		}
	}
	return w.writeUserAgents(userAgents.UserAgents)
}

func (w *workbook) writeSummary(campaign *domain.Campaign, period domain.Period, summary *domain.CampaignViewSummary, listed int64, generatedAt time.Time) error {
	s, err := w.stream(summarySheet, []float64{28, 60})
	if err != nil {
		return err
	}

	periodEnd := func(t time.Time, open string) any {
		if t.IsZero() {
			return open
		}
		return excelize.Cell{StyleID: w.dateTime, Value: t}
	}
	label := func(text string) excelize.Cell { return excelize.Cell{StyleID: w.header, Value: text} }
	rows := [][]any{
		{label("Campanha"), campaign.Name},
		{label("Id da campanha"), campaign.Id},
		{label("De (UTC)"), periodEnd(period.FromUtc, "Desde a primeira visualização")},
		{label("Até (UTC, exclusive)"), periodEnd(period.ToUtc, "Até a última visualização")},
		{label("Visualizações totais"), summary.TotalViews},
		{label("Visualizações únicas"), summary.UniqueViews},
		{label("Gerado em (UTC)"), excelize.Cell{StyleID: w.dateTime, Value: generatedAt}},
		{},
		{excelize.Cell{StyleID: w.wrap, Value: "Visualizações únicas: por destinatário (externalUserId) ou, sem ele, por IP. " +
			"Nas abas por cliente, sistema e dispositivo, a contagem única é feita dentro de cada linha."}},
	}
	if listed < summary.TotalViews {
		rows = append(rows, []any{excelize.Cell{StyleID: w.wrap, Value: fmt.Sprintf(
			"A aba Visualizações traz as %d mais recentes de %d: é o limite de linhas do Excel.", listed, summary.TotalViews)}})
	}
	for _, row := range rows {
		if err := s.add(row...); err != nil {
			return err
		}
	}
	return s.flush()
}

func (w *workbook) writeDaily(days []domain.DailyCampaignViews) error {
	s, err := w.stream(dailySheet, []float64{14, 14, 14}, "Dia (UTC)", "Total", "Únicas")
	if err != nil {
		return err
	}
	for _, d := range days {
		if err := s.add(excelize.Cell{StyleID: w.date, Value: d.DateUtc}, d.TotalViews, d.UniqueViews); err != nil {
			return err
		}
	}
	return s.flush()
}

func (w *workbook) writeGroups(name, column string, groups []domain.ViewGroup, total int64) error {
	s, err := w.stream(name, []float64{32, 14, 14, 14}, column, "Total", "Únicas", "% do total")
	if err != nil {
		return err
	}
	for _, g := range groups {
		share := 0.0
		if total > 0 {
			share = float64(g.TotalViews) / float64(total)
		}
		if err := s.add(userAgentLabel(g.Name), g.TotalViews, g.UniqueViews, excelize.Cell{StyleID: w.percent, Value: share}); err != nil {
			return err
		}
	}
	return s.flush()
}

func (w *workbook) writeUserAgents(userAgents []domain.UserAgentViews) error {
	s, err := w.stream(userAgentsSheet, []float64{80, 24, 20, 18, 12, 12},
		"User agent", "Cliente de e-mail / navegador", "Sistema operacional", "Dispositivo", "Total", "Únicas")
	if err != nil {
		return err
	}
	for _, ua := range userAgents {
		raw := ua.UserAgent
		if raw == "" {
			raw = "(não informado)"
		}
		if err := s.add(raw, userAgentLabel(ua.Client), userAgentLabel(ua.OS), userAgentLabel(ua.Device),
			ua.TotalViews, ua.UniqueViews); err != nil {
			return err
		}
	}
	return s.flush()
}

// writeViews lists every view, up to the rows a sheet holds, and returns how
// many were listed.
func (w *workbook) writeViews(eachView func(func(domain.CampaignView) error) error) (int64, error) {
	s, err := w.stream(viewsSheet, []float64{20, 30, 18, 80, 24, 20, 18},
		"Quando (UTC)", "Destinatário (externalUserId)", "IP", "User agent", "Cliente de e-mail / navegador", "Sistema operacional", "Dispositivo")
	if err != nil {
		return 0, err
	}
	var listed int64
	parsed := map[string]useragent.Info{}
	err = eachView(func(v domain.CampaignView) error {
		if listed == maxSheetViews {
			return errSheetFull
		}
		info, ok := parsed[v.UserAgent]
		if !ok {
			info = useragent.Parse(v.UserAgent)
			parsed[v.UserAgent] = info
		}
		listed++
		return s.add(excelize.Cell{StyleID: w.dateTime, Value: v.ViewedOnUtc}, v.ExternalUserId, v.IpAddress, v.UserAgent,
			userAgentLabel(info.Client), userAgentLabel(info.OS), userAgentLabel(info.Device))
	})
	if err != nil && !errors.Is(err, errSheetFull) {
		return 0, err
	}
	return listed, s.flush()
}
