import type { Metadata } from "next";

import { Pagination } from "@/components/pagination";
import { Alert, buttonClass, Card, ColorSwatch, EmptyState, inputClass, LinkButton, PageHeader, Table, Td } from "@/components/ui";
import { api, ApiError, getOrNotFound } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import { formatDateTime, formatDay, formatNumber, pageParam, pixelColor } from "@/lib/format";
import { translate } from "@/lib/forms";
import type { Campaign, CampaignView, DailyViews, Page, UserAgentReport, ViewSummary } from "@/lib/types";
import { CampaignStatusBadge } from "../status";
import { CopySnippet } from "./copy-snippet";
import { DailyChart } from "./daily-chart";
import { reportPeriod } from "./period";
import { UserAgentGroups, UserAgentTable } from "./user-agents";

export const metadata: Metadata = { title: "Relatórios da campanha" };

// Public address of the pixel, as the e-mail recipients reach the API.
const PIXEL_BASE_URL = (process.env.PIXEL_BASE_URL ?? "http://localhost:8080").replace(/\/+$/, "");

type Reports = [ViewSummary, DailyViews, UserAgentReport, Page<CampaignView>];

export default async function CampaignPage({ params, searchParams }: PageProps<"/campaigns/[id]">) {
  const me = await requirePermission((m) => m.permissions.campaigns.read);
  const { id } = await params;
  const query = await searchParams;

  const { today, from, to, query: period } = reportPeriod(query);
  const viewsPage = pageParam(query.page);
  const exportError = typeof query.exportError === "string" ? query.exportError : undefined;

  const path = `/api/campaigns/${encodeURIComponent(id)}`;
  const campaign = await getOrNotFound<Campaign>(path);

  let reports: Reports | undefined;
  let periodError: string | undefined;
  try {
    reports = await Promise.all([
      api<ViewSummary>("GET", `${path}/reports/summary`, { query: period }),
      api<DailyViews>("GET", `${path}/reports/daily`, { query: period }),
      api<UserAgentReport>("GET", `${path}/reports/user-agents`, { query: period }),
      api<Page<CampaignView>>("GET", `${path}/views`, { query: { ...period, page: viewsPage, pageSize: 20 } }),
    ]);
  } catch (err) {
    if (!(err instanceof ApiError) || err.status !== 400) {
      throw err;
    }
    periodError = err.fields.length ? err.fields.map((f) => translate(f.message)).join(" ") : translate(err.message);
    if (/366 days/.test(periodError)) {
      periodError = "O relatório diário cobre no máximo 366 dias. Diminua o período.";
    }
  }

  const snippet = `<img src="${PIXEL_BASE_URL}/pixel/${campaign.id}?externalUserId=ID_DO_DESTINATARIO" width="1" height="1" alt="">`;

  return (
    <>
      <PageHeader
        title={campaign.name}
        description={
          <span className="flex flex-wrap items-center gap-2">
            <CampaignStatusBadge campaign={campaign} />
            <span>Expira em {formatDateTime(campaign.expireInUtc)}</span>
            <span className="flex items-center gap-1.5">
              · <ColorSwatch color={pixelColor(campaign)} size="h-4 w-4" />
              <span className="font-mono text-xs">{pixelColor(campaign)}</span>
            </span>
          </span>
        }
        actions={
          me.permissions.campaigns.write && (
            <LinkButton href={`/campaigns/${campaign.id}/edit`} variant="secondary">
              Editar
            </LinkButton>
          )
        }
      />

      <Card className="mb-6 p-5">
        <h2 className="text-sm font-semibold text-zinc-900">Pixel para o e-mail</h2>
        <p className="mb-3 mt-1 text-sm text-zinc-500">
          Coloque no HTML do e-mail e troque <code className="font-mono text-xs">ID_DO_DESTINATARIO</code> pelo identificador do
          destinatário no seu sistema. Sem ele, as visualizações únicas são contadas pelo IP.
        </p>
        <CopySnippet text={snippet} />
      </Card>

      <form className="mb-6 flex flex-wrap items-end gap-3" aria-label="Período do relatório">
        <label className="text-sm font-medium text-zinc-800">
          De
          <input type="date" name="from" defaultValue={from} max={today} className={`${inputClass()} mt-1`} />
        </label>
        <label className="text-sm font-medium text-zinc-800">
          Até
          <input type="date" name="to" defaultValue={to} className={`${inputClass()} mt-1`} />
        </label>
        <button type="submit" className={buttonClass("secondary")}>
          Aplicar
        </button>
        <p className="pb-2 text-xs text-zinc-500">Dias em UTC.</p>
        {reports && (
          // A plain link: the route handler answers a file, not a page.
          <a
            href={`/campaigns/${campaign.id}/export?${new URLSearchParams({ from, to })}`}
            className={`${buttonClass("primary")} sm:ml-auto`}
            download
          >
            Exportar Excel
          </a>
        )}
      </form>

      {exportError && (
        <div className="mb-6">
          <Alert>
            Não foi possível exportar a planilha. Tente novamente.
            {exportError !== "1" && <span className="mt-1 block text-xs">Código da requisição: {exportError}</span>}
          </Alert>
        </div>
      )}

      {periodError || !reports ? (
        <Alert>{periodError}</Alert>
      ) : (
        <Reports campaignId={campaign.id} reports={reports} params={{ from, to }} />
      )}
    </>
  );
}

function Reports({
  campaignId,
  reports: [summary, daily, userAgents, views],
  params,
}: {
  campaignId: string;
  reports: Reports;
  params: Record<string, string>;
}) {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2">
        <Card className="p-5">
          <p className="text-sm text-zinc-500">Visualizações totais</p>
          <p className="mt-2 text-3xl font-semibold tabular-nums text-zinc-900">{formatNumber(summary.totalViews)}</p>
        </Card>
        <Card className="p-5">
          <p className="text-sm text-zinc-500">Visualizações únicas</p>
          <p className="mt-2 text-3xl font-semibold tabular-nums text-zinc-900">{formatNumber(summary.uniqueViews)}</p>
          <p className="mt-1 text-xs text-zinc-500">Por destinatário (externalUserId) ou, sem ele, por IP.</p>
        </Card>
      </div>

      <Card className="p-5">
        <h2 className="mb-4 text-sm font-semibold text-zinc-900">Por dia</h2>
        {daily.days.length === 0 ? (
          <EmptyState title="Nenhuma visualização no período" />
        ) : (
          <>
            <DailyChart days={daily.days} />
            <details className="mt-2">
              <summary className="cursor-pointer text-sm font-medium text-indigo-600">Ver tabela</summary>
              <div className="mt-3 max-h-80 overflow-y-auto rounded-md border border-zinc-200">
                <Table head={["Dia", "Total", "Únicas"]}>
                  {daily.days.map((d) => (
                    <tr key={d.date}>
                      <Td>{formatDay(d.date)}</Td>
                      <Td className="tabular-nums">{formatNumber(d.totalViews)}</Td>
                      <Td className="tabular-nums">{formatNumber(d.uniqueViews)}</Td>
                    </tr>
                  ))}
                </Table>
              </div>
            </details>
          </>
        )}
      </Card>

      {summary.totalViews > 0 && (
        <>
          <UserAgentGroups report={userAgents} />
          <UserAgentTable userAgents={userAgents.userAgents} />
        </>
      )}

      <Card>
        <h2 className="px-5 pt-5 text-sm font-semibold text-zinc-900">Visualizações</h2>
        <p className="px-5 pb-4 text-xs text-zinc-500">Das mais recentes para as mais antigas.</p>
        {views.items.length === 0 ? (
          <EmptyState title="Nenhuma visualização no período" />
        ) : (
          <Table head={["Quando", "Destinatário", "IP", "User-Agent"]}>
            {views.items.map((v) => (
              <tr key={v.id}>
                <Td>{formatDateTime(v.viewedOnUtc)}</Td>
                <Td className="font-mono text-xs">{v.externalUserId ?? <span className="text-zinc-400">—</span>}</Td>
                <Td className="font-mono text-xs">{v.ipAddress}</Td>
                <Td className="max-w-xs truncate text-xs" >
                  <span title={v.userAgent}>{v.userAgent ?? <span className="text-zinc-400">—</span>}</span>
                </Td>
              </tr>
            ))}
          </Table>
        )}
        <Pagination page={views.page} pageSize={views.pageSize} total={views.total} path={`/campaigns/${campaignId}`} params={params} />
      </Card>
    </div>
  );
}
