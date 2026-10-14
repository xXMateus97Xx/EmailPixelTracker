export default function Loading() {
  return (
    <div className="animate-pulse space-y-4" aria-busy="true" aria-label="Carregando">
      <div className="h-8 w-48 rounded bg-zinc-200" />
      <div className="h-4 w-80 rounded bg-zinc-200" />
      <div className="h-64 rounded-lg bg-zinc-200/70" />
    </div>
  );
}
