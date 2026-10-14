import { LinkButton } from "@/components/ui";

export default function NotFound() {
  return (
    <main className="flex min-h-[60vh] flex-col items-center justify-center gap-3 px-4 text-center">
      <p className="text-sm font-semibold text-indigo-600">404</p>
      <h1 className="text-xl font-semibold text-zinc-900">Página não encontrada</h1>
      <p className="text-sm text-zinc-500">O registro não existe ou pertence a outro tenant.</p>
      <LinkButton href="/" variant="secondary" className="mt-2">
        Voltar ao início
      </LinkButton>
    </main>
  );
}
