"use client";

import { useState } from "react";

import { formatDay, formatNumber } from "@/lib/format";
import { TOTAL, UNIQUE } from "./colors";

type Day = { date: string; totalViews: number; uniqueViews: number };

/** Rounds the axis maximum up to 1, 2 or 5 × 10ⁿ, so the gridlines get round values. */
function niceMax(value: number): number {
  if (value <= 4) {
    return 4;
  }
  const magnitude = 10 ** Math.floor(Math.log10(value));
  const step = [1, 2, 5, 10].find((s) => s * magnitude >= value)!;
  return step * magnitude;
}

/**
 * Views per day: unique views (a subset of the total) drawn inside the total
 * bar, on one axis. Hover or focus a day for its numbers; the same data is in
 * the table below the chart.
 */
export function DailyChart({ days }: { days: Day[] }) {
  const [hover, setHover] = useState<number | null>(null);
  const max = niceMax(Math.max(...days.map((d) => d.totalViews), 0));
  const ticks = [max, max / 2, 0];
  const gap = days.length > 90 ? "gap-px" : "gap-0.5";
  const labelIndexes = new Set([0, Math.floor((days.length - 1) / 2), days.length - 1]);
  const active = hover === null ? null : days[hover];

  return (
    <figure aria-label="Visualizações por dia" className="select-none">
      <div className="mb-3 flex items-center justify-between gap-4 text-xs text-zinc-600">
        <div className="flex gap-4">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ background: TOTAL }} aria-hidden /> Total
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ background: UNIQUE }} aria-hidden /> Únicas
          </span>
        </div>
        <p aria-live="polite" className="tabular-nums text-zinc-700">
          {active
            ? `${formatDay(active.date)}: ${formatNumber(active.totalViews)} total, ${formatNumber(active.uniqueViews)} únicas`
            : "Passe o mouse sobre um dia"}
        </p>
      </div>

      <div className="flex gap-2">
        <div className="flex h-48 flex-col justify-between text-right text-[11px] tabular-nums text-zinc-400" aria-hidden>
          {ticks.map((t) => (
            <span key={t} className="-translate-y-1/2 first:translate-y-0 last:translate-y-0">
              {formatNumber(t)}
            </span>
          ))}
        </div>
        <div className="relative h-48 flex-1">
          {/* Recessive gridlines. */}
          {ticks.map((t) => (
            <div key={t} className="absolute inset-x-0 border-t border-zinc-100" style={{ bottom: `${(t / max) * 100}%` }} aria-hidden />
          ))}
          <div className={`absolute inset-0 flex items-end ${gap}`} onMouseLeave={() => setHover(null)}>
            {days.map((d, i) => (
              // The whole column is the hit target, not just the bar.
              <button
                key={d.date}
                type="button"
                className={`relative flex h-full min-w-0 flex-1 items-end justify-center rounded-t outline-none focus-visible:bg-indigo-50 ${
                  hover === i ? "bg-zinc-100" : ""
                }`}
                onMouseEnter={() => setHover(i)}
                onFocus={() => setHover(i)}
                onBlur={() => setHover(null)}
                aria-label={`${formatDay(d.date)}: ${d.totalViews} total, ${d.uniqueViews} únicas`}
              >
                <span className="absolute bottom-0 w-full rounded-t" style={{ height: `${(d.totalViews / max) * 100}%`, background: TOTAL }} />
                <span className="absolute bottom-0 w-1/2 rounded-t" style={{ height: `${(d.uniqueViews / max) * 100}%`, background: UNIQUE }} />
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="mt-1.5 flex pl-8 text-[11px] text-zinc-400" aria-hidden>
        {days.map((d, i) => (
          <span key={d.date} className="relative min-w-0 flex-1">
            {labelIndexes.has(i) && (
              <span
                className={`absolute whitespace-nowrap ${i === 0 ? "left-0" : i === days.length - 1 ? "right-0" : "left-1/2 -translate-x-1/2"}`}
              >
                {formatDay(d.date)}
              </span>
            )}
          </span>
        ))}
      </div>
      <div className="h-4" />
    </figure>
  );
}
