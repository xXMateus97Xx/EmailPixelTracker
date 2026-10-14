"use client";

import { buttonClass } from "@/components/ui";

// Messages of server errors are hidden in production; the digest links the
// error to the server log.
export default function PanelError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <div className="mx-auto max-w-md py-16 text-center">
      <h1 className="text-lg font-semibold text-zinc-900">Não foi possível carregar esta página</h1>
      <p className="mt-2 text-sm text-zinc-500">
        A API pode estar indisponível. Tente novamente em instantes.
        {error.digest && <span className="mt-1 block font-mono text-xs">Código: {error.digest}</span>}
      </p>
      <button type="button" onClick={reset} className={buttonClass("secondary", "mt-6")}>
        Tentar novamente
      </button>
    </div>
  );
}
