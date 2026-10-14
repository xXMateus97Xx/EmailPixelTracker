"use client";

import { useActionState } from "react";

import { Checkbox, Field, FormError, SubmitButton, TextInput } from "@/components/form";
import { Card, LinkButton } from "@/components/ui";
import { emptyForm, type FormAction } from "@/lib/form-state";
import type { Tenant } from "@/lib/types";

export function TenantForm({ tenant, save }: { tenant?: Tenant; save: FormAction }) {
  const [state, action] = useActionState(save, emptyForm);
  const v = state.values;
  return (
    <Card className="max-w-xl">
      <form action={action} className="space-y-5 p-6">
        <FormError state={state} />
        <Field label="Nome" name="name" error={state.fields?.name} hint="Único entre todos os tenants.">
          <TextInput name="name" required maxLength={200} defaultValue={v?.name ?? tenant?.name} error={state.fields?.name} />
        </Field>
        <Checkbox
          name="active"
          label="Ativo"
          description="Usuários de um tenant inativo não conseguem entrar, e o pixel das campanhas dele deixa de contar."
          defaultChecked={v ? v.active === "on" : (tenant?.active ?? true)}
        />
        <div className="flex gap-2 pt-2">
          <SubmitButton>{tenant ? "Salvar" : "Criar tenant"}</SubmitButton>
          <LinkButton href="/tenants" variant="secondary">
            Cancelar
          </LinkButton>
        </div>
      </form>
    </Card>
  );
}
