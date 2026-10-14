import type { Metadata } from "next";

import { PageHeader } from "@/components/ui";
import { getOrNotFound } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import type { Tenant } from "@/lib/types";
import { saveTenant } from "../actions";
import { TenantForm } from "../tenant-form";

export const metadata: Metadata = { title: "Editar tenant" };

export default async function EditTenantPage({ params }: PageProps<"/tenants/[id]">) {
  await requirePermission((m) => m.permissions.tenants.write);
  const { id } = await params;
  const tenant = await getOrNotFound<Tenant>(`/api/tenants/${encodeURIComponent(id)}`);
  return (
    <>
      <PageHeader title="Editar tenant" description={tenant.name} />
      <TenantForm tenant={tenant} save={saveTenant.bind(null, tenant.id)} />
    </>
  );
}
