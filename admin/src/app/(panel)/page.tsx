import Link from "next/link";

import { Alert, Card, LinkButton, PageHeader } from "@/components/ui";
import { api } from "@/lib/api";
import { getMe } from "@/lib/auth";
import { formatNumber, roleLabels } from "@/lib/format";
import type { Page } from "@/lib/types";

export default async function HomePage() {
  const me = await getMe();
  const { permissions } = me;

  // Only the totals are needed, so each list asks for a single item.
  const count = (path: string) => api<Page<unknown>>("GET", path, { query: { pageSize: 1 } }).then((p) => p.total);
  const [tenants, users, campaigns] = await Promise.all([
    permissions.tenants.read ? count("/api/tenants") : undefined,
    permissions.users.read ? count("/api/users") : undefined,
    permissions.campaigns.read ? count("/api/campaigns") : undefined,
  ]);

  const stats = [
    { label: "Tenants", value: tenants, href: "/tenants" },
    { label: me.tenant ? "Usuários do tenant" : "SuperAdmins", value: users, href: "/users" },
    { label: "Campanhas", value: campaigns, href: "/campaigns" },
  ].filter((s) => s.value !== undefined);

  return (
    <>
      <PageHeader
        title={`Olá, ${me.user.name.split(" ")[0]}`}
        description={`${roleLabels[me.user.role].label}${me.tenant ? ` em ${me.tenant.name}` : ""}. ${roleLabels[me.user.role].description}`}
      />

      {me.user.role === "SuperAdmin" && !me.impersonatedTenantId && (
        <div className="mb-6">
          <Alert tone="indigo">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span>Campanhas e usuários pertencem a um tenant. Escolha um tenant para acessá-lo.</span>
              <LinkButton href="/tenants" variant="secondary" className="py-1">
                Escolher tenant
              </LinkButton>
            </div>
          </Alert>
        </div>
      )}

      {stats.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-3">
          {stats.map((s) => (
            <Link key={s.href} href={s.href} className="group">
              <Card className="p-5 transition-colors group-hover:border-indigo-300">
                <p className="text-sm text-zinc-500">{s.label}</p>
                <p className="mt-2 text-3xl font-semibold tabular-nums text-zinc-900">{formatNumber(s.value!)}</p>
              </Card>
            </Link>
          ))}
        </div>
      )}

      {permissions.campaigns.write && (
        <div className="mt-8">
          <LinkButton href="/campaigns/new">Nova campanha</LinkButton>
        </div>
      )}
    </>
  );
}
