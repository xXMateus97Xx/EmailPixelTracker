package api

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
	"github.com/xXMateus97Xx/email-pixel-tracker/telemetry"
)

// pixelCache keeps the encoded 1x1 PNG of each color in memory, keyed by its
// RGBA. Colors only come from campaigns, so the cache grows at most with the
// number of distinct campaign colors.
type pixelCache struct {
	images  sync.Map // color.NRGBA -> []byte
	metrics *telemetry.Metrics
}

func (p *pixelCache) get(campaign *domain.Campaign) ([]byte, error) {
	key := color.NRGBA{
		R: toByte(campaign.PixelColorR),
		G: toByte(campaign.PixelColorG),
		B: toByte(campaign.PixelColorB),
		A: toByte(int(math.Round(float64(campaign.PixelColorA) * 255))),
	}
	if cached, ok := p.images.Load(key); ok {
		return cached.([]byte), nil
	}

	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, key)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	// Concurrent misses for the same color keep the first stored image.
	cached, loaded := p.images.LoadOrStore(key, buf.Bytes())
	if !loaded {
		p.metrics.PixelCached()
	}
	return cached.([]byte), nil
}

// pixelResult is the result label of a pixel request in the metrics and logs.
func pixelResult(err error) string {
	switch {
	case err == nil:
		return "tracked"
	case errors.Is(err, service.ErrCampaignNotFound):
		return "campaign_not_found"
	case errors.Is(err, service.ErrCampaignInactive):
		return "campaign_inactive"
	case errors.Is(err, service.ErrCampaignExpired):
		return "campaign_expired"
	case errors.Is(err, service.ErrCampaignTenantInactive):
		return "tenant_inactive"
	case errors.Is(err, service.ErrValidation):
		return "invalid_request"
	default:
		return "error"
	}
}

func toByte(v int) uint8 {
	return uint8(min(max(v, 0), 255))
}
