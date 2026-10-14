"use client";

import { useActionState, useState } from "react";

import { Checkbox, Field, FormError, SubmitButton, TextInput } from "@/components/form";
import { Card, inputClass, LinkButton } from "@/components/ui";
import { roleLabels } from "@/lib/format";
import { emptyForm, type FormAction } from "@/lib/form-state";
import type { Role, RoleInfo, User } from "@/lib/types";

export function UserForm({
  user,
  roles,
  isSelf,
  save,
}: {
  user?: User;
  roles: RoleInfo[];
  isSelf: boolean;
  save: FormAction;
}) {
  const [state, action] = useActionState(save, emptyForm);
  const v = state.values;
  const f = state.fields ?? {};

  // The current role stays selectable even if it could not be assigned now.
  const options = roles.filter((r) => r.assignable || r.name === user?.role);
  const initialRole = (v?.role as Role | undefined) ?? user?.role ?? options.find((r) => r.assignable)?.name;
  const [role, setRole] = useState(initialRole);

  return (
    <Card className="max-w-xl">
      <form action={action} className="space-y-5 p-6">
        <FormError state={state} />
        <Field label="Nome" name="name" error={f.name} hint="Nome de exibição.">
          <TextInput name="name" required maxLength={200} defaultValue={v?.name ?? user?.name} error={f.name} />
        </Field>
        <Field label="Usuário" name="username" error={f.username} hint="Usado para entrar. Sem espaços; maiúsculas e minúsculas são iguais.">
          <TextInput
            name="username"
            required
            maxLength={50}
            pattern="\S+"
            autoCapitalize="none"
            spellCheck={false}
            autoComplete="off"
            defaultValue={v?.username ?? user?.username}
            error={f.username}
          />
        </Field>
        <Field
          label={user ? "Nova senha" : "Senha"}
          name="password"
          error={f.password}
          hint={user ? "Deixe em branco para manter a senha atual." : "Pelo menos 8 caracteres."}
        >
          <TextInput name="password" type="password" required={!user} minLength={8} autoComplete="new-password" error={f.password} />
        </Field>
        <Field label="Role" name="role" error={f.role} hint={role ? roleLabels[role].description : undefined}>
          <select
            id="role"
            name="role"
            defaultValue={initialRole}
            onChange={(e) => setRole(e.target.value as Role)}
            required
            className={inputClass(!!f.role)}
          >
            {options.map((r) => (
              <option key={r.name} value={r.name} disabled={!r.assignable}>
                {roleLabels[r.name].label}
              </option>
            ))}
          </select>
        </Field>
        {isSelf ? (
          <>
            {/* Users cannot deactivate themselves. */}
            <input type="hidden" name="active" value="on" />
            <Checkbox name="active-display" label="Ativo" description="Você não pode desativar o próprio usuário." defaultChecked disabled />
          </>
        ) : (
          <Checkbox
            name="active"
            label="Ativo"
            description="Usuários inativos não conseguem entrar, e sessões abertas deixam de funcionar na hora."
            defaultChecked={v ? v.active === "on" : (user?.active ?? true)}
          />
        )}
        <div className="flex gap-2 pt-2">
          <SubmitButton>{user ? "Salvar" : "Criar usuário"}</SubmitButton>
          <LinkButton href="/users" variant="secondary">
            Cancelar
          </LinkButton>
        </div>
      </form>
    </Card>
  );
}
