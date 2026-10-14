package repository

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRepositoriesAreLazyAndCreatedOnce(t *testing.T) {
	var tenantsCalls, usersCalls, campaignsCalls atomic.Int32
	repos := NewRepositories(Constructors{
		Tenants:       func() TenantRepository { tenantsCalls.Add(1); return nil },
		Users:         func() UserRepository { usersCalls.Add(1); return nil },
		Campaigns:     func() CampaignRepository { campaignsCalls.Add(1); return nil },
		CampaignViews: func() CampaignViewRepository { return nil },
	})

	if tenantsCalls.Load()+usersCalls.Load()+campaignsCalls.Load() != 0 {
		t.Fatal("repositories were created before being used")
	}

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { repos.Tenants() })
	}
	wg.Wait()

	if got := tenantsCalls.Load(); got != 1 {
		t.Fatalf("tenants constructor called %d times, want 1", got)
	}
	if usersCalls.Load() != 0 || campaignsCalls.Load() != 0 {
		t.Fatal("unused repositories were created")
	}
}
