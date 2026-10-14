import type { Metadata } from "next";

import { PageHeader } from "@/components/ui";
import { getOrNotFound } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import type { Campaign } from "@/lib/types";
import { saveCampaign } from "../../actions";
import { CampaignForm } from "../../campaign-form";

export const metadata: Metadata = { title: "Editar campanha" };

export default async function EditCampaignPage({ params }: PageProps<"/campaigns/[id]/edit">) {
  await requirePermission((m) => m.permissions.campaigns.write);
  const { id } = await params;
  const campaign = await getOrNotFound<Campaign>(`/api/campaigns/${encodeURIComponent(id)}`);
  return (
    <>
      <PageHeader title="Editar campanha" description={campaign.name} />
      <CampaignForm campaign={campaign} defaultExpire={campaign.expireInUtc} save={saveCampaign.bind(null, campaign.id)} />
    </>
  );
}
