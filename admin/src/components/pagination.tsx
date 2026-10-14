import Link from "next/link";

import { buttonClass } from "./ui";

/** Previous/next links that keep the other query parameters of the page. */
export function Pagination({
  page,
  pageSize,
  total,
  path,
  params = {},
}: {
  page: number;
  pageSize: number;
  total: number;
  path: string;
  params?: Record<string, string | undefined>;
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  if (total === 0) {
    return null;
  }

  const href = (p: number) => {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value) {
        query.set(key, value);
      }
    }
    query.set("page", String(p));
    return `${path}?${query}`;
  };

  const first = (page - 1) * pageSize + 1;
  const last = Math.min(page * pageSize, total);
  const disabled = "pointer-events-none opacity-50";

  return (
    <nav aria-label="Paginação" className="flex items-center justify-between border-t border-zinc-200 px-4 py-3 text-sm">
      <p className="text-zinc-500">
        {first}–{last} de {total}
      </p>
      <div className="flex gap-2">
        <Link
          href={href(page - 1)}
          aria-disabled={page <= 1}
          tabIndex={page <= 1 ? -1 : undefined}
          className={buttonClass("secondary", page <= 1 ? disabled : "")}
        >
          Anterior
        </Link>
        <Link
          href={href(page + 1)}
          aria-disabled={page >= pages}
          tabIndex={page >= pages ? -1 : undefined}
          className={buttonClass("secondary", page >= pages ? disabled : "")}
        >
          Próxima
        </Link>
      </div>
    </nav>
  );
}
