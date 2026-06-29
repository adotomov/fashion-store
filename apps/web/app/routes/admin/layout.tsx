import { Outlet, useMatches } from "react-router";

import { AdminHeader } from "../../components/admin/AdminHeader";
import { AdminSidebar } from "../../components/admin/AdminSidebar";
import { RequireAdmin } from "../../components/RequireAdmin";

export default function AdminLayout() {
  const matches = useMatches();
  const title = (matches.at(-1)?.handle as { title?: string } | undefined)?.title ?? "Dashboard";

  return (
    <RequireAdmin>
      <div className="flex h-screen overflow-hidden bg-stone-50">
        <AdminSidebar />
        <div className="flex h-full flex-1 flex-col overflow-hidden">
          <AdminHeader title={title} />
          <main className="flex-1 overflow-y-auto p-8">
            <Outlet />
          </main>
        </div>
      </div>
    </RequireAdmin>
  );
}
