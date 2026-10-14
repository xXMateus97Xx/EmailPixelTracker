import type { Campaign, Role } from "./types";

export const roleLabels: Record<Role, { label: string; description: string }> = {
  SuperAdmin: {
    label: "SuperAdmin",
    description: "Usuário mestre, sem tenant. Gerencia tenants e acessa qualquer tenant.",
  },
  Admin: { label: "Admin", description: "Gerencia usuários e campanhas do tenant." },
  ContentManager: { label: "Gestor de conteúdo", description: "Gerencia as campanhas do tenant." },
  API: { label: "API", description: "Somente leitura de campanhas e relatórios, para integrações." },
};

const dateTime = new Intl.DateTimeFormat("pt-BR", {
  dateStyle: "short",
  timeStyle: "short",
  timeZone: "UTC",
});

const dateOnly = new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeZone: "UTC" });

/** Dates are shown in UTC, like the API stores them. */
export function formatDateTime(iso: string): string {
  return `${dateTime.format(new Date(iso))} UTC`;
}

/** "2026-10-14" → "14/10/2026" */
export function formatDay(day: string): string {
  return dateOnly.format(new Date(`${day}T00:00:00Z`));
}

export function formatNumber(n: number): string {
  return n.toLocaleString("pt-BR");
}

const percent = new Intl.NumberFormat("pt-BR", { style: "percent", maximumFractionDigits: 1 });

export function formatPercent(part: number, total: number): string {
  return percent.format(total > 0 ? part / total : 0);
}

// Generic names of GET /api/campaigns/{id}/reports/user-agents; product
// names (Gmail, Windows...) are shown as they are.
const userAgentLabels: Record<string, string> = {
  Unknown: "Desconhecido",
  Other: "Outro",
  Desktop: "Computador",
  Mobile: "Celular",
  Tablet: "Tablet",
  "Email proxy": "Proxy de e-mail",
  Bot: "Robô / automação",
};

export function userAgentLabel(name: string): string {
  return userAgentLabels[name] ?? name;
}

export function pixelColor(c: Pick<Campaign, "pixelColorR" | "pixelColorG" | "pixelColorB" | "pixelColorA">): string {
  return `rgba(${c.pixelColorR}, ${c.pixelColorG}, ${c.pixelColorB}, ${c.pixelColorA})`;
}

export type CampaignStatus = "active" | "inactive" | "expired";

export function campaignStatus(c: Campaign, now = new Date()): CampaignStatus {
  if (!c.active) {
    return "inactive";
  }
  return new Date(c.expireInUtc) <= now ? "expired" : "active";
}

const DAY_MS = 24 * 60 * 60 * 1000;

/** "YYYY-MM-DD" of the UTC day `offsetDays` from today (or from `day`). */
export function utcDay(offsetDays = 0, day?: string): string {
  const base = day ? Date.parse(`${day}T00:00:00Z`) : Date.now();
  return new Date(base + offsetDays * DAY_MS).toISOString().slice(0, 10);
}

/** datetime-local value (UTC) `days` from now. */
export function dateTimeFromNow(days: number): string {
  return toDateTimeLocal(new Date(Date.now() + days * DAY_MS).toISOString());
}

/** Value for <input type="datetime-local">, in UTC. */
export function toDateTimeLocal(iso: string): string {
  return iso.slice(0, 16);
}

/** Positive integer from a search param, or the fallback. */
export function pageParam(value: string | string[] | undefined, fallback = 1): number {
  const n = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(n) && n > 0 ? n : fallback;
}
