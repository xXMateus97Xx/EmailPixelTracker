import type { Metadata } from "next";

import { RecaptchaNotice } from "@/components/recaptcha-notice";
import { PageHeader } from "@/components/ui";
import { getMe } from "@/lib/auth";
import { recaptchaFor } from "@/lib/recaptcha";
import { PasswordForm } from "./password-form";

export const metadata: Metadata = { title: "Alterar senha" };

export default async function PasswordPage() {
  const [me, recaptcha] = await Promise.all([getMe(), recaptchaFor("changePassword")]);
  return (
    <>
      <PageHeader title="Alterar senha" description={`A senha que você usa para entrar como @${me.user.username}.`} />
      <PasswordForm username={me.user.username} recaptcha={recaptcha} />
      {recaptcha && <RecaptchaNotice className="mt-4" />}
    </>
  );
}
