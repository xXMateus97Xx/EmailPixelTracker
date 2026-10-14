import type { Metadata } from "next";

import { PageHeader } from "@/components/ui";
import { requirePermission } from "@/lib/auth";
import { dateTimeFromNow } from "@/lib/format";
import { saveCampaign } from "../actions";
import { CampaignForm } from "../campaign-form";

export const metadata: Metadata = { title: "Nova campanha" };

export default async function NewCampaignPage() {
  const me = await requirePermission((m) => m.permissions.campaigns.write);
  return (
    <>
      <PageHeader title="Nova campanha" description={`No tenant ${me.tenant?.name ?? ""}.`} />
      {/* Suggested expiration: 30 days from now. */}
      <CampaignForm defaultExpire={dateTimeFromNow(30)} save={saveCampaign.bind(null, undefined)} />
    </>
  );
}
