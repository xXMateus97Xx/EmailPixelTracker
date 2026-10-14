package service

import (
	"cmp"
	"slices"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/useragent"
)

// viewCounter counts the views of a group and its distinct visitors.
type viewCounter struct {
	total    int64
	visitors map[string]struct{}
}

func (c *viewCounter) add(visitor string, views int64) {
	c.total += views
	c.visitors[visitor] = struct{}{}
}

func (c *viewCounter) summary() domain.CampaignViewSummary {
	return domain.CampaignViewSummary{TotalViews: c.total, UniqueViews: int64(len(c.visitors))}
}

// groupCounter counts the views per group name. A visitor seen with two user
// agents of the same group (e.g. Chrome 128 and Chrome 129) is one unique view
// of that group.
type groupCounter map[string]*viewCounter

func (g groupCounter) add(name, visitor string, views int64) {
	c := g[name]
	if c == nil {
		c = &viewCounter{visitors: map[string]struct{}{}}
		g[name] = c
	}
	c.add(visitor, views)
}

func (g groupCounter) groups() []domain.ViewGroup {
	groups := make([]domain.ViewGroup, 0, len(g))
	for name, c := range g {
		groups = append(groups, domain.ViewGroup{Name: name, CampaignViewSummary: c.summary()})
	}
	slices.SortFunc(groups, func(a, b domain.ViewGroup) int {
		return compareMostViewed(a.CampaignViewSummary, b.CampaignViewSummary, a.Name, b.Name)
	})
	return groups
}

// compareMostViewed orders by total views, then unique views (both
// descending), then name, so that the order is stable.
func compareMostViewed(a, b domain.CampaignViewSummary, aName, bName string) int {
	return cmp.Or(
		cmp.Compare(b.TotalViews, a.TotalViews),
		cmp.Compare(b.UniqueViews, a.UniqueViews),
		cmp.Compare(aName, bName),
	)
}

func buildUserAgentReport(rows []domain.VisitorUserAgentViews) *domain.UserAgentReport {
	clients, systems, devices, raw := groupCounter{}, groupCounter{}, groupCounter{}, groupCounter{}
	parsed := map[string]useragent.Info{}
	for _, row := range rows {
		info, ok := parsed[row.UserAgent]
		if !ok {
			info = useragent.Parse(row.UserAgent)
			parsed[row.UserAgent] = info
		}
		clients.add(info.Client, row.Visitor, row.Views)
		systems.add(info.OS, row.Visitor, row.Views)
		devices.add(info.Device, row.Visitor, row.Views)
		raw.add(row.UserAgent, row.Visitor, row.Views)
	}

	userAgents := make([]domain.UserAgentViews, 0, len(raw))
	for _, g := range raw.groups() {
		info := parsed[g.Name]
		userAgents = append(userAgents, domain.UserAgentViews{
			UserAgent:           g.Name,
			Client:              info.Client,
			OS:                  info.OS,
			Device:              info.Device,
			CampaignViewSummary: g.CampaignViewSummary,
		})
	}
	return &domain.UserAgentReport{
		Clients:          clients.groups(),
		OperatingSystems: systems.groups(),
		Devices:          devices.groups(),
		UserAgents:       userAgents,
	}
}
