import type { Metadata } from "next";
import Link from "next/link";

import { Pagination } from "@/components/pagination";
import { ActiveBadge, Badge, Card, EmptyState, LinkButton, PageHeader, Table, Td } from "@/components/ui";
import { api } from "@/lib/api";
import { requirePermission } from "@/lib/auth";
import { formatDateTime, pageParam, roleLabels } from "@/lib/format";
import type { Page, User } from "@/lib/types";

export const metadata: Metadata = { title: "Usuários" };

export default async function UsersPage({ searchParams }: PageProps<"/users">) {
  const me = await requirePermission((m) => m.permissions.users.read);
  const page = pageParam((await searchParams).page);
  const users = await api<Page<User>>("GET", "/api/users", { query: { page, pageSize: 20 } });

  const description = me.tenant
    ? `Usuários do tenant ${me.tenant.name}.`
    : "Usuários SuperAdmin. Para gerenciar os usuários de um tenant, acesse o tenant.";

  return (
    <>
      <PageHeader
        title={me.tenant ? "Usuários" : "SuperAdmins"}
        description={description}
        actions={me.permissions.users.write && <LinkButton href="/users/new">Novo usuário</LinkButton>}
      />
      <Card>
        {users.items.length === 0 ? (
          <EmptyState title="Nenhum usuário cadastrado" />
        ) : (
          <Table head={["Nome", "Usuário", "Role", "Status", "Criado em", ""]}>
            {users.items.map((u) => (
              <tr key={u.id}>
                <Td className="font-medium text-zinc-900">
                  {u.name} {u.id === me.user.id && <Badge>Você</Badge>}
                </Td>
                <Td className="font-mono text-xs">@{u.username}</Td>
                <Td>{roleLabels[u.role].label}</Td>
                <Td>
                  <ActiveBadge active={u.active} />
                </Td>
                <Td>{formatDateTime(u.createdAtUtc)}</Td>
                <Td className="text-right">
                  {me.permissions.users.write && (
                    <Link href={`/users/${u.id}`} className="rounded-md px-2 py-1 text-sm font-medium text-indigo-600 hover:bg-indigo-50">
                      Editar
                    </Link>
                  )}
                </Td>
              </tr>
            ))}
          </Table>
        )}
        <Pagination page={users.page} pageSize={users.pageSize} total={users.total} path="/users" />
      </Card>
    </>
  );
}
