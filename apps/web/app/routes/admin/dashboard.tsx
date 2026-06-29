import { useSearchParams } from "react-router";

import { CatalogTab } from "../../components/admin/dashboard/CatalogTab";
import { OrdersTab } from "../../components/admin/dashboard/OrdersTab";
import { UsersTab } from "../../components/admin/dashboard/UsersTab";
import { Tabs } from "../../components/admin/Tabs";

export const handle = { title: "Dashboard" };

const tabs = [
  { id: "orders", label: "Orders" },
  { id: "catalog", label: "Catalog" },
  { id: "users", label: "Users" },
];

const validTabIds = new Set(tabs.map((t) => t.id));

export default function AdminDashboard() {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedTab = searchParams.get("tab");
  const activeTab = requestedTab && validTabIds.has(requestedTab) ? requestedTab : "orders";

  function handleChange(tabId: string) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("tab", tabId);
      return next;
    });
  }

  return (
    <div className="flex flex-col gap-6">
      <Tabs tabs={tabs} activeTab={activeTab} onChange={handleChange} />

      {activeTab === "orders" && <OrdersTab />}
      {activeTab === "catalog" && <CatalogTab />}
      {activeTab === "users" && <UsersTab />}
    </div>
  );
}
