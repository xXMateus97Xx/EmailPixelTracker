import type { Metadata } from "next";

import { Alert } from "@/components/ui";
import { getRecaptchaConfig } from "@/lib/recaptcha";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Entrar" };

const reasons: Record<string, string> = {
  expired: "Sua sessão expirou. Entre novamente.",
  "tenant-inactive": "O tenant da sua conta está inativo.",
};

export default async function LoginPage({ searchParams }: PageProps<"/login">) {
  const { reason } = await searchParams;
  const message = typeof reason === "string" ? reasons[reason] : undefined;
  const recaptcha = await getRecaptchaConfig();

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
          <LoginForm recaptcha={recaptcha.enabled && recaptcha.siteKey ? { siteKey: recaptcha.siteKey, action: recaptcha.action ?? "login" } : undefined} />
        </div>
        {recaptcha.enabled && (
          // Google's terms allow hiding the badge only with this notice.
          <p className="mt-4 text-center text-xs text-zinc-500">
            Protegido pelo reCAPTCHA. Aplicam-se a{" "}
            <a href="https://policies.google.com/privacy" className="underline hover:text-zinc-700" target="_blank" rel="noreferrer">
              Política de Privacidade
            </a>{" "}
            e os{" "}
            <a href="https://policies.google.com/terms" className="underline hover:text-zinc-700" target="_blank" rel="noreferrer">
              Termos de Serviço
            </a>{" "}
            do Google.
          </p>
        )}
      </div>
    </main>
  );
}
