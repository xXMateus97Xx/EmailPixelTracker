import type { Metadata } from "next";
import Link from "next/link";

import { SubmitButton } from "@/components/form";
import { Pagination } from "@/components/pagination";
import { ActiveBadge, Badge, Card, EmptyState, LinkButton, PageHeader, Table, Td } from "@/components/ui";
import { api } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import { formatDateTime, pageParam } from "@/lib/format";
import type { Page, Tenant } from "@/lib/types";
import { impersonate } from "../actions";

export const metadata: Metadata = { title: "Tenants" };

export default async function TenantsPage({ searchParams }: PageProps<"/tenants">) {
  const me = await requirePermission((m) => m.permissions.tenants.read);
  const page = pageParam((await searchParams).page);
  const tenants = await api<Page<Tenant>>("GET", "/api/tenants", { query: { page, pageSize: 20 } });

  return (
    <>
      <PageHeader
        title="Tenants"
        description="Clientes do sistema. Acesse um tenant para gerenciar as campanhas e os usuários dele."
        actions={me.permissions.tenants.write && <LinkButton href="/tenants/new">Novo tenant</LinkButton>}
      />
      <Card>
        {tenants.items.length === 0 ? (
          <EmptyState title="Nenhum tenant cadastrado">Crie o primeiro tenant para começar.</EmptyState>
        ) : (
          <Table head={["Nome", "Status", "Criado em", ""]}>
            {tenants.items.map((t) => {
              const current = t.id === me.impersonatedTenantId;
              return (
                <tr key={t.id}>
                  <Td className="font-medium text-zinc-900">
                    {t.name} {current && <Badge tone="indigo">Acessando</Badge>}
                  </Td>
                  <Td>
                    <ActiveBadge active={t.active} />
                  </Td>
                  <Td>{formatDateTime(t.createdAtUtc)}</Td>
                  <Td className="text-right">
                    <div className="flex justify-end gap-2">
                      {me.permissions.tenants.write && (
                        <Link href={`/tenants/${t.id}`} className="rounded-md px-2 py-1 text-sm font-medium text-indigo-600 hover:bg-indigo-50">
                          Editar
                        </Link>
                      )}
                      {me.permissions.impersonate && !current && (
                        <form action={impersonate.bind(null, t.id)}>
                          <SubmitButton variant="secondary" pendingText="Acessando…" className="py-1">
                            {t.active ? "Acessar" : "Consultar"}
                          </SubmitButton>
                        </form>
                      )}
                    </div>
                  </Td>
                </tr>
              );
            })}
          </Table>
        )}
        <Pagination page={tenants.page} pageSize={tenants.pageSize} total={tenants.total} path="/tenants" />
      </Card>
    </>
  );
}
