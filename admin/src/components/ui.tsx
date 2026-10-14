import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";

// Small presentational building blocks shared by the pages (server-safe).

type Variant = "primary" | "secondary" | "danger" | "ghost";

const variants: Record<Variant, string> = {
  primary: "bg-indigo-600 text-white hover:bg-indigo-500 focus-visible:outline-indigo-600 disabled:bg-indigo-300",
  secondary:
    "bg-white text-zinc-800 ring-1 ring-inset ring-zinc-300 hover:bg-zinc-50 focus-visible:outline-indigo-600 disabled:text-zinc-400",
  danger: "bg-red-600 text-white hover:bg-red-500 focus-visible:outline-red-600 disabled:bg-red-300",
  ghost: "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 focus-visible:outline-indigo-600",
};

export function buttonClass(variant: Variant = "primary", extra = ""): string {
  return [
    "inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium shadow-xs",
    "focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed transition-colors",
    variants[variant],
    extra,
  ].join(" ");
}

export function inputClass(invalid?: boolean): string {
  return [
    "block w-full rounded-md bg-white px-3 py-2 text-sm text-zinc-900 shadow-xs ring-1 ring-inset",
    "placeholder:text-zinc-400 focus:outline-2 focus:-outline-offset-1 focus:outline-indigo-600",
    "disabled:bg-zinc-50 disabled:text-zinc-500",
    invalid ? "ring-red-400" : "ring-zinc-300",
  ].join(" ");
}

export function LinkButton({
  variant = "primary",
  className = "",
  ...props
}: ComponentProps<typeof Link> & { variant?: Variant }) {
  return <Link className={buttonClass(variant, className)} {...props} />;
}

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">{title}</h1>
        {description && <p className="mt-1 text-sm text-zinc-500">{description}</p>}
      </div>
      {actions && <div className="flex shrink-0 gap-2">{actions}</div>}
    </div>
  );
}

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-lg border border-zinc-200 bg-white shadow-xs ${className}`}>{children}</div>;
}

type Tone = "green" | "red" | "amber" | "zinc" | "indigo";

const tones: Record<Tone, string> = {
  green: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
  red: "bg-red-50 text-red-700 ring-red-600/20",
  amber: "bg-amber-50 text-amber-800 ring-amber-600/20",
  zinc: "bg-zinc-100 text-zinc-700 ring-zinc-500/20",
  indigo: "bg-indigo-50 text-indigo-700 ring-indigo-600/20",
};

export function Badge({ tone = "zinc", children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${tones[tone]}`}>
      {children}
    </span>
  );
}

export function ActiveBadge({ active }: { active: boolean }) {
  return active ? <Badge tone="green">Ativo</Badge> : <Badge tone="red">Inativo</Badge>;
}

export function Alert({ tone = "red", children }: { tone?: "red" | "amber" | "indigo"; children: ReactNode }) {
  const styles = {
    red: "border-red-200 bg-red-50 text-red-800",
    amber: "border-amber-200 bg-amber-50 text-amber-900",
    indigo: "border-indigo-200 bg-indigo-50 text-indigo-900",
  };
  return (
    <div role={tone === "red" ? "alert" : "status"} className={`rounded-md border px-4 py-3 text-sm ${styles[tone]}`}>
      {children}
    </div>
  );
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="px-6 py-12 text-center">
      <p className="text-sm font-medium text-zinc-900">{title}</p>
      {children && <div className="mt-1 text-sm text-zinc-500">{children}</div>}
    </div>
  );
}

export function Table({ head, children }: { head: ReactNode[]; children: ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-zinc-200 text-sm">
        <thead className="bg-zinc-50">
          <tr>
            {head.map((h, i) => (
              <th key={i} scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-zinc-500">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-100 bg-white">{children}</tbody>
      </table>
    </div>
  );
}

export function Td({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <td className={`whitespace-nowrap px-4 py-3 text-zinc-700 ${className}`}>{children}</td>;
}

export function ColorSwatch({ color, size = "h-5 w-5" }: { color: string; size?: string }) {
  // The checkerboard shows through transparent colors.
  return (
    <span
      className={`inline-block shrink-0 rounded border border-zinc-300 ${size}`}
      style={{
        backgroundImage: `linear-gradient(${color}, ${color}), repeating-conic-gradient(#e4e4e7 0% 25%, #fff 0% 50%)`,
        backgroundSize: "100% 100%, 8px 8px",
      }}
      aria-hidden
    />
  );
}
