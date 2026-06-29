import { apiFetch } from "./client";

export type InventoryItem = {
  id: string;
  variant_id: string;
  sku: string;
  quantity_on_hand: number;
  quantity_reserved: number;
  quantity_available: number;
  product_name?: string;
  variant_label?: string;
  created_at: string;
  updated_at: string;
};

export type AdminAdjustableMovementType = "admin_adjustment" | "return" | "manual_correction";

export type InventoryMovement = {
  id: string;
  type: string;
  quantity_delta: number;
  note: string;
  created_by?: string;
  created_at: string;
};

export function listInventoryItems(): Promise<InventoryItem[]> {
  return apiFetch<InventoryItem[]>("/api/v1/admin/inventory/items");
}

export function getInventoryItem(id: string): Promise<InventoryItem> {
  return apiFetch<InventoryItem>(`/api/v1/admin/inventory/items/${id}`);
}

export function createInventoryItem(variantId: string, sku: string, initialQuantity: number): Promise<InventoryItem> {
  return apiFetch<InventoryItem>("/api/v1/admin/inventory/items", {
    method: "POST",
    body: { variant_id: variantId, sku, initial_quantity: initialQuantity },
  });
}

export function updateInventorySKU(id: string, sku: string): Promise<InventoryItem> {
  return apiFetch<InventoryItem>(`/api/v1/admin/inventory/items/${id}`, { method: "PATCH", body: { sku } });
}

export function adjustStock(
  id: string,
  type: AdminAdjustableMovementType,
  quantityDelta: number,
  note: string,
): Promise<InventoryItem> {
  return apiFetch<InventoryItem>(`/api/v1/admin/inventory/items/${id}/adjust`, {
    method: "POST",
    body: { type, quantity_delta: quantityDelta, note },
  });
}

export function listMovements(itemId: string): Promise<InventoryMovement[]> {
  return apiFetch<InventoryMovement[]>(`/api/v1/admin/inventory/items/${itemId}/movements`);
}
