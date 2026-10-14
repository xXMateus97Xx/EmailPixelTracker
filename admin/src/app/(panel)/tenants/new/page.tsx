import type { Metadata } from "next";

import { PageHeader } from "@/components/ui";
import { requirePermission } from "@/lib/auth";
import { saveTenant } from "../actions";
import { TenantForm } from "../tenant-form";

export const metadata: Metadata = { title: "Novo tenant" };

export default async function NewTenantPage() {
  await requirePermission((m) => m.permissions.tenants.write);
  return (
    <>
      <PageHeader title="Novo tenant" />
      <TenantForm save={saveTenant.bind(null, undefined)} />
    </>
  );
}
