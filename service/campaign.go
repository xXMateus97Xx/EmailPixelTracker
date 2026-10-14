package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	campaignNameMaxLength = 200
)

type CampaignInput struct {
	Name        string
	Active      bool
	ExpireInUtc time.Time
	PixelColorR int
	PixelColorG int
	PixelColorB int
	PixelColorA float32
}

type CampaignService struct {
	authn     authenticator
	campaigns repository.CampaignRepository
}

func (s *CampaignService) Create(ctx context.Context, input CampaignInput) (*domain.Campaign, error) {
	tenantId, err := s.authn.tenantScope(ctx, campaignWriters, true)
	if err != nil {
		return nil, err
	}

	campaign := &domain.Campaign{
		Id:           newId(),
		TenantId:     tenantId,
		CreatedAtUtc: nowUtc(),
	}
	applyCampaignInput(campaign, input)

	if err := s.validate(ctx, campaign, true); err != nil {
		return nil, err
	}
	if err := s.campaigns.Create(ctx, campaign); err != nil {
		return nil, mapRepoError(err)
	}
	return campaign, nil
}

func (s *CampaignService) Update(ctx context.Context, id string, input CampaignInput) (*domain.Campaign, error) {
	tenantId, err := s.authn.tenantScope(ctx, campaignWriters, true)
	if err != nil {
		return nil, err
	}

	campaign, err := s.campaigns.GetById(ctx, tenantId, id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	applyCampaignInput(campaign, input)

	if err := s.validate(ctx, campaign, false); err != nil {
		return nil, err
	}
	if err := s.campaigns.Update(ctx, tenantId, campaign); err != nil {
		return nil, mapRepoError(err)
	}
	return campaign, nil
}

func (s *CampaignService) GetById(ctx context.Context, id string) (*domain.Campaign, error) {
	tenantId, err := s.authn.tenantScope(ctx, campaignReaders, false)
	if err != nil {
		return nil, err
	}
	campaign, err := s.campaigns.GetById(ctx, tenantId, id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return campaign, nil
}

func (s *CampaignService) List(ctx context.Context, page Page) (*PagedResult[domain.Campaign], error) {
	tenantId, err := s.authn.tenantScope(ctx, campaignReaders, false)
	if err != nil {
		return nil, err
	}
	items, total, err := s.campaigns.List(ctx, tenantId, page.toPagination())
	if err != nil {
		return nil, mapRepoError(err)
	}
	return newPagedResult(items, total, page), nil
}

func applyCampaignInput(campaign *domain.Campaign, input CampaignInput) {
	campaign.Name = strings.TrimSpace(input.Name)
	campaign.Active = input.Active
	campaign.ExpireInUtc = input.ExpireInUtc.UTC()
	campaign.PixelColorR = input.PixelColorR
	campaign.PixelColorG = input.PixelColorG
	campaign.PixelColorB = input.PixelColorB
	campaign.PixelColorA = input.PixelColorA
}

func (s *CampaignService) validate(ctx context.Context, campaign *domain.Campaign, creating bool) error {
	v := &validator{}

	nameLength := utf8.RuneCountInString(campaign.Name)
	v.check(nameLength >= 1 && nameLength <= campaignNameMaxLength, "name", "must have between 1 and 200 characters")

	switch {
	case campaign.ExpireInUtc.IsZero():
		v.add("expireInUtc", "is required")
	case creating && !campaign.ExpireInUtc.After(nowUtc()):
		v.add("expireInUtc", "must be in the future")
	case !campaign.ExpireInUtc.After(campaign.CreatedAtUtc):
		v.add("expireInUtc", "must be after the campaign creation date")
	}

	v.check(isColorChannel(campaign.PixelColorR), "pixelColorR", "must be between 0 and 255")
	v.check(isColorChannel(campaign.PixelColorG), "pixelColorG", "must be between 0 and 255")
	v.check(isColorChannel(campaign.PixelColorB), "pixelColorB", "must be between 0 and 255")
	v.check(campaign.PixelColorA >= 0 && campaign.PixelColorA <= 1, "pixelColorA", "must be between 0 and 1")

	if err := v.err(); err != nil {
		return err
	}

	taken, err := s.campaigns.ExistsByName(ctx, campaign.TenantId, campaign.Name, campaign.Id)
	if err != nil {
		return err
	}
	v.check(!taken, "name", "is already in use")
	return v.err()
}

func isColorChannel(value int) bool {
	return value >= 0 && value <= 255
}
