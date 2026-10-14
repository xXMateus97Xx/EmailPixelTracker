import { SubmitButton } from "@/components/form";
import { Badge } from "@/components/ui";
import { getMe } from "@/lib/auth";
import { roleLabels } from "@/lib/format";
import { logout, stopImpersonation } from "./actions";
import { Nav, type NavItem } from "./nav";

export default async function PanelLayout({ children }: LayoutProps<"/">) {
  const me = await getMe();
  const { permissions } = me;

  const items: NavItem[] = [{ href: "/", label: "Início" }];
  if (permissions.tenants.read) items.push({ href: "/tenants", label: "Tenants" });
  if (permissions.users.read) items.push({ href: "/users", label: "Usuários" });
  if (permissions.campaigns.read) items.push({ href: "/campaigns", label: "Campanhas" });

  return (
    <div className="min-h-screen lg:flex">
      <aside className="border-b border-zinc-200 bg-zinc-100 lg:fixed lg:inset-y-0 lg:w-60 lg:border-r lg:border-b-0">
        <div className="flex h-full flex-col gap-4 p-4">
          <div className="flex items-center gap-2 px-1">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-600" aria-hidden>
              <span className="h-2 w-2 rounded-xs bg-white" />
            </span>
            <span className="text-sm font-semibold text-zinc-900">Email Pixel Tracker</span>
          </div>
          <Nav items={items} />
          <div className="mt-auto hidden border-t border-zinc-200 pt-4 lg:block">
            <UserCard me={me} />
          </div>
        </div>
      </aside>

      <div className="min-w-0 flex-1 lg:pl-60">
        {me.impersonatedTenantId && (
          <div className={`border-b px-4 py-2.5 text-sm lg:px-8 ${me.readOnly ? "border-amber-200 bg-amber-50" : "border-indigo-200 bg-indigo-50"}`}>
            <form action={stopImpersonation} className="flex flex-wrap items-center justify-between gap-2">
              <p className={me.readOnly ? "text-amber-900" : "text-indigo-900"}>
                Acessando o tenant <strong>{me.tenant?.name}</strong> como SuperAdmin
                {me.readOnly && " — tenant inativo, somente consulta"}.
              </p>
              <SubmitButton variant="secondary" pendingText="Saindo…" className="py-1">
                Sair do tenant
              </SubmitButton>
            </form>
          </div>
        )}
        <div className="flex items-center justify-end border-b border-zinc-200 bg-white px-4 py-2 lg:hidden">
          <UserCard me={me} compact />
        </div>
        <main className="mx-auto max-w-6xl px-4 py-8 lg:px-8">{children}</main>
      </div>
    </div>
  );
}

function UserCard({ me, compact = false }: { me: Awaited<ReturnType<typeof getMe>>; compact?: boolean }) {
  return (
    <div className={compact ? "flex items-center gap-3" : "space-y-3 px-1"}>
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-zinc-900">{me.user.name}</p>
        <p className="truncate text-xs text-zinc-500">
          @{me.user.username}
          {me.tenant && !me.impersonatedTenantId && ` · ${me.tenant.name}`}
        </p>
        {!compact && (
          <div className="mt-2">
            <Badge tone="indigo">{roleLabels[me.user.role].label}</Badge>
          </div>
        )}
      </div>
      <form action={logout}>
        <SubmitButton variant="secondary" pendingText="Saindo…" className={compact ? "py-1" : "w-full"}>
          Sair
        </SubmitButton>
      </form>
    </div>
  );
}
