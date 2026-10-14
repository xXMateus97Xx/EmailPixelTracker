import { Card, Table, Td } from "@/components/ui";
import { formatNumber, formatPercent, userAgentLabel } from "@/lib/format";
import type { UserAgentReport, ViewGroup } from "@/lib/types";
import { TOTAL, UNIQUE } from "./colors";

/** User agents listed on the page; the spreadsheet has all of them. */
const TABLE_LIMIT = 20;

/** The views by e-mail client, operating system and device, side by side. */
export function UserAgentGroups({ report }: { report: UserAgentReport }) {
  const dimensions = [
    { title: "Cliente de e-mail ou navegador", groups: report.clients },
    { title: "Sistema operacional", groups: report.operatingSystems },
    { title: "Dispositivo", groups: report.devices },
  ];
  return (
    <Card className="p-5">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold text-zinc-900">Por user agent</h2>
        <div className="flex gap-4 text-xs text-zinc-600">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ background: TOTAL }} aria-hidden /> Total
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ background: UNIQUE }} aria-hidden /> Únicas
          </span>
        </div>
      </div>
      <div className="grid gap-8 lg:grid-cols-3">
        {dimensions.map((d) => (
          <section key={d.title} aria-label={d.title} className="min-w-0">
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-zinc-500">{d.title}</h3>
            <BarList groups={d.groups} />
          </section>
        ))}
      </div>
      <p className="mt-5 text-xs text-zinc-500">
        Gmail e Yahoo Mail baixam as imagens por um proxy, que esconde o sistema e o dispositivo de quem abriu. As visualizações
        únicas são contadas dentro de cada linha: quem abriu no celular e no computador conta uma vez em cada um.
      </p>
    </Card>
  );
}

/** One row per group: unique views drawn in front of the total, on a shared scale. */
function BarList({ groups }: { groups: ViewGroup[] }) {
  const max = Math.max(...groups.map((g) => g.totalViews), 1);
  const total = groups.reduce((sum, g) => sum + g.totalViews, 0);
  return (
    <ul className="space-y-3">
      {groups.map((g) => (
        <li key={g.name}>
          <div className="flex items-baseline justify-between gap-3 text-sm">
            <span className="truncate text-zinc-800" title={userAgentLabel(g.name)}>
              {userAgentLabel(g.name)}
            </span>
            <span className="shrink-0 tabular-nums text-zinc-500">
              <span className="font-medium text-zinc-900">{formatNumber(g.totalViews)}</span> · {formatNumber(g.uniqueViews)} únicas ·{" "}
              {formatPercent(g.totalViews, total)}
            </span>
          </div>
          <div className="relative mt-1.5 h-2 rounded-full bg-zinc-100" aria-hidden>
            <span className="absolute inset-y-0 left-0 rounded-full" style={{ width: `${(g.totalViews / max) * 100}%`, background: TOTAL }} />
            <span className="absolute inset-y-0 left-0 rounded-full" style={{ width: `${(g.uniqueViews / max) * 100}%`, background: UNIQUE }} />
          </div>
        </li>
      ))}
    </ul>
  );
}

export function UserAgentTable({ userAgents }: { userAgents: UserAgentReport["userAgents"] }) {
  const shown = userAgents.slice(0, TABLE_LIMIT);
  return (
    <Card>
      <h2 className="px-5 pt-5 text-sm font-semibold text-zinc-900">User agents</h2>
      <p className="px-5 pb-4 text-xs text-zinc-500">
        {userAgents.length > shown.length
          ? `Os ${shown.length} mais vistos de ${formatNumber(userAgents.length)}. A planilha exportada traz todos.`
          : "Do mais visto para o menos visto."}
      </p>
      <Table head={["User agent", "Cliente", "Sistema", "Dispositivo", "Total", "Únicas"]}>
        {shown.map((ua) => (
          <tr key={ua.userAgent}>
            <Td className="max-w-xs truncate font-mono text-xs">
              {ua.userAgent ? <span title={ua.userAgent}>{ua.userAgent}</span> : <span className="text-zinc-400">(não informado)</span>}
            </Td>
            <Td>{userAgentLabel(ua.client)}</Td>
            <Td>{userAgentLabel(ua.os)}</Td>
            <Td>{userAgentLabel(ua.device)}</Td>
            <Td className="tabular-nums">{formatNumber(ua.totalViews)}</Td>
            <Td className="tabular-nums">{formatNumber(ua.uniqueViews)}</Td>
          </tr>
        ))}
      </Table>
    </Card>
  );
}
