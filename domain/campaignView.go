package domain

import "time"

type CampaignView struct {
	Id          string
	CampaignId  string
	ViewedOnUtc time.Time
	// ExternalUserId is empty when the pixel URL did not inform it.
	ExternalUserId string
	IpAddress      string
	// UserAgent is empty when the client did not send one.
	UserAgent string
}

// Period filters by time, from FromUtc (inclusive) to ToUtc (exclusive).
// A zero value leaves that side unbounded.
type Period struct {
	FromUtc time.Time
	ToUtc   time.Time
}

// CampaignViewSummary counts views. A unique view is a distinct
// ExternalUserId or, for views without it, a distinct IpAddress.
type CampaignViewSummary struct {
	TotalViews  int64
	UniqueViews int64
}

type DailyCampaignViews struct {
	// DateUtc is the day (UTC, at midnight) the views happened.
	DateUtc time.Time
	CampaignViewSummary
}

// VisitorUserAgentViews counts the views of one visitor with one user agent.
// Visitor is the key that identifies a unique view (see CampaignViewSummary);
// it is only compared, never shown.
type VisitorUserAgentViews struct {
	UserAgent string
	Visitor   string
	Views     int64
}

// ViewGroup counts the views of one value of a report dimension, such as the
// "Gmail" client or the "iOS" operating system.
type ViewGroup struct {
	Name string
	CampaignViewSummary
}

// UserAgentViews counts the views of one raw user agent, with what it was
// classified as.
type UserAgentViews struct {
	// UserAgent is empty for the views without one.
	UserAgent string
	Client    string
	OS        string
	Device    string
	CampaignViewSummary
}

// UserAgentReport groups the views by what the user agent tells. Every list
// is ordered from the most viewed.
type UserAgentReport struct {
	Clients          []ViewGroup
	OperatingSystems []ViewGroup
	Devices          []ViewGroup
	UserAgents       []UserAgentViews
}
