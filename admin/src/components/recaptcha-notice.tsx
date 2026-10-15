/** Google's terms allow hiding the reCAPTCHA badge only with this notice. */
export function RecaptchaNotice({ className = "" }: { className?: string }) {
  return (
    <p className={`text-xs text-zinc-500 ${className}`}>
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
  );
}
