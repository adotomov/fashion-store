import { EmptyState } from "../../components/admin/EmptyState";

export const handle = { title: "Invoices & Tax" };

export default function AdminInvoices() {
  return (
    <EmptyState
      icon="invoices"
      title="No invoices yet"
      description="Invoice records and tax configuration will live here once the invoicing backend module is built."
    />
  );
}
