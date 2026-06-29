import { useSearchParams } from "react-router";

import { AttributesTab } from "../../components/admin/catalog/AttributesTab";
import { CatalogsTab } from "../../components/admin/catalog/CatalogsTab";
import { CategoriesTab } from "../../components/admin/catalog/CategoriesTab";
import { ProductsTab } from "../../components/admin/catalog/ProductsTab";
import { ProductTypesTab } from "../../components/admin/catalog/ProductTypesTab";
import { Tabs } from "../../components/admin/Tabs";

export const handle = { title: "Catalog" };

const tabs = [
  { id: "products", label: "Products" },
  { id: "types", label: "Types" },
  { id: "categories", label: "Categories" },
  { id: "attributes", label: "Attributes" },
  { id: "catalogs", label: "Catalogs" },
];

const validTabIds = new Set(tabs.map((t) => t.id));

export default function AdminCatalog() {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedTab = searchParams.get("tab");
  const activeTab = requestedTab && validTabIds.has(requestedTab) ? requestedTab : "products";

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

      {activeTab === "products" && <ProductsTab />}
      {activeTab === "types" && <ProductTypesTab />}
      {activeTab === "categories" && <CategoriesTab />}
      {activeTab === "attributes" && <AttributesTab />}
      {activeTab === "catalogs" && <CatalogsTab />}
    </div>
  );
}
