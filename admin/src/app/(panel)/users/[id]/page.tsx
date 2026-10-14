import type { Metadata } from "next";

import { PageHeader } from "@/components/ui";
import { api, getOrNotFound } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import type { RoleInfo, User } from "@/lib/types";
import { saveUser } from "../actions";
import { UserForm } from "../user-form";

export const metadata: Metadata = { title: "Editar usuário" };

export default async function EditUserPage({ params }: PageProps<"/users/[id]">) {
  const me = await requirePermission((m) => m.permissions.users.write);
  const { id } = await params;
  const [user, roles] = await Promise.all([
    getOrNotFound<User>(`/api/users/${encodeURIComponent(id)}`),
    api<{ items: RoleInfo[] }>("GET", "/api/roles"),
  ]);
  return (
    <>
      <PageHeader title="Editar usuário" description={`${user.name} (@${user.username})`} />
      <UserForm user={user} roles={roles.items} isSelf={user.id === me.user.id} save={saveUser.bind(null, user.id)} />
    </>
  );
}
