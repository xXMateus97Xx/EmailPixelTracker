import type { Metadata } from "next";

import { RecaptchaNotice } from "@/components/recaptcha-notice";
import { Alert } from "@/components/ui";
import { recaptchaFor } from "@/lib/recaptcha";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Entrar" };

const reasons: Record<string, string> = {
  expired: "Sua sessão expirou. Entre novamente.",
  "tenant-inactive": "O tenant da sua conta está inativo.",
};

export default async function LoginPage({ searchParams }: PageProps<"/login">) {
  const { reason } = await searchParams;
  const message = typeof reason === "string" ? reasons[reason] : undefined;
  const recaptcha = await recaptchaFor("login");

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-12">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-600">
            <span className="h-3 w-3 rounded-sm bg-white" aria-hidden />
          </div>
          <h1 className="text-xl font-semibold text-zinc-900">Email Pixel Tracker</h1>
          <p className="mt-1 text-sm text-zinc-500">Painel de administração</p>
        </div>
        <div className="space-y-5 rounded-lg border border-zinc-200 bg-white p-6 shadow-xs">
          {message && <Alert tone="amber">{message}</Alert>}
          <LoginForm recaptcha={recaptcha} />
        </div>
        {recaptcha && <RecaptchaNotice className="mt-4 text-center" />}
      </div>
    </main>
  );
}
