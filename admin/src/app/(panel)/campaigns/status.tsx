import { Badge } from "@/components/ui";
import { campaignStatus } from "@/lib/format";
import type { Campaign } from "@/lib/types";

export function CampaignStatusBadge({ campaign }: { campaign: Campaign }) {
  switch (campaignStatus(campaign)) {
    case "active":
      return <Badge tone="green">Ativa</Badge>;
    case "expired":
      return <Badge tone="amber">Expirada</Badge>;
    default:
      return <Badge tone="red">Inativa</Badge>;
  }
}
