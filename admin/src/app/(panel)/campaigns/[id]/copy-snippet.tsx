"use client";

import { useState } from "react";

import { buttonClass } from "@/components/ui";

export function CopySnippet({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
      <pre className="min-w-0 flex-1 overflow-x-auto rounded-md bg-zinc-900 px-3 py-2.5 font-mono text-xs leading-relaxed text-zinc-100">
        {text}
      </pre>
      <button
        type="button"
        className={buttonClass("secondary")}
        onClick={async () => {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 2000);
        }}
      >
        {copied ? "Copiado!" : "Copiar"}
      </button>
    </div>
  );
}
