import type { Metadata } from "next";
import Link from "next/link";

import { Pagination } from "@/components/pagination";
import { Card, ColorSwatch, EmptyState, LinkButton, PageHeader, Table, Td } from "@/components/ui";
import { api } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import { formatDateTime, pageParam, pixelColor } from "@/lib/format";
import type { Campaign, Page } from "@/lib/types";
import { CampaignStatusBadge } from "./status";

export const metadata: Metadata = { title: "Campanhas" };

export default async function CampaignsPage({ searchParams }: PageProps<"/campaigns">) {
  const me = await requirePermission((m) => m.permissions.campaigns.read);
  const page = pageParam((await searchParams).page);
  const campaigns = await api<Page<Campaign>>("GET", "/api/campaigns", { query: { page, pageSize: 20 } });
  const canWrite = me.permissions.campaigns.write;

  return (
    <>
      <PageHeader
        title="Campanhas"
        description={`Campanhas do tenant ${me.tenant?.name ?? ""}. Cada uma tem um pixel para os e-mails.`}
        actions={canWrite && <LinkButton href="/campaigns/new">Nova campanha</LinkButton>}
      />
      <Card>
        {campaigns.items.length === 0 ? (
          <EmptyState title="Nenhuma campanha cadastrada">
            {canWrite ? "Crie uma campanha para gerar o pixel." : "As campanhas criadas aparecerão aqui."}
          </EmptyState>
        ) : (
          <Table head={["Nome", "Pixel", "Status", "Expira em", ""]}>
            {campaigns.items.map((c) => (
              <tr key={c.id}>
                <Td className="font-medium text-zinc-900">
                  <Link href={`/campaigns/${c.id}`} className="hover:text-indigo-600">
                    {c.name}
                  </Link>
                </Td>
                <Td>
                  <ColorSwatch color={pixelColor(c)} />
                </Td>
                <Td>
                  <CampaignStatusBadge campaign={c} />
                </Td>
                <Td>{formatDateTime(c.expireInUtc)}</Td>
                <Td className="text-right">
                  <div className="flex justify-end gap-1">
                    <Link href={`/campaigns/${c.id}`} className="rounded-md px-2 py-1 text-sm font-medium text-indigo-600 hover:bg-indigo-50">
                      Relatórios
                    </Link>
                    {canWrite && (
                      <Link href={`/campaigns/${c.id}/edit`} className="rounded-md px-2 py-1 text-sm font-medium text-indigo-600 hover:bg-indigo-50">
                        Editar
                      </Link>
                    )}
                  </div>
                </Td>
              </tr>
            ))}
          </Table>
        )}
        <Pagination page={campaigns.page} pageSize={campaigns.pageSize} total={campaigns.total} path="/campaigns" />
      </Card>
    </>
  );
}
