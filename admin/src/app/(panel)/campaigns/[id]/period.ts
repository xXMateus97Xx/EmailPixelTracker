import { utcDay } from "@/lib/format";

const isDate = (s: unknown): s is string => typeof s === "string" && /^\d{4}-\d{2}-\d{2}$/.test(s) && !isNaN(Date.parse(s));

type SearchParams = Record<string, string | string[] | undefined>;

/**
 * The report period of ?from=&to=: whole UTC days, both ends included, the
 * last 30 days by default. `query` is what the API takes: [fromUtc, toUtc).
 */
export function reportPeriod(params: SearchParams) {
  const today = utcDay();
  const from = isDate(params.from) ? params.from : utcDay(-29);
  const to = isDate(params.to) ? params.to : today;
  return { today, from, to, query: { fromUtc: `${from}T00:00:00Z`, toUtc: `${utcDay(1, to)}T00:00:00Z` } };
}
