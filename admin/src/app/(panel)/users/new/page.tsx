import type { Metadata } from "next";

import { PageHeader } from "@/components/ui";
import { api } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import type { RoleInfo } from "@/lib/types";
import { saveUser } from "../actions";
import { UserForm } from "../user-form";

export const metadata: Metadata = { title: "Novo usuário" };

export default async function NewUserPage() {
  const me = await requirePermission((m) => m.permissions.users.write);
  const roles = await api<{ items: RoleInfo[] }>("GET", "/api/roles");
  return (
    <>
      <PageHeader title="Novo usuário" description={me.tenant ? `No tenant ${me.tenant.name}.` : "Novo SuperAdmin, sem tenant."} />
      <UserForm roles={roles.items} isSelf={false} save={saveUser.bind(null, undefined)} />
    </>
  );
}
