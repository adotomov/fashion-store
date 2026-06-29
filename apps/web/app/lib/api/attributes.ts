import { apiFetch } from "./client";

export type AttributeValue = {
  id: string;
  attribute_id: string;
  value: string;
};

export type Attribute = {
  id: string;
  name: string;
  values: AttributeValue[];
  created_at: string;
  updated_at: string;
};

export function listAttributes(): Promise<Attribute[]> {
  return apiFetch<Attribute[]>("/api/v1/admin/attributes");
}

export function createAttribute(name: string): Promise<Attribute> {
  return apiFetch<Attribute>("/api/v1/admin/attributes", { method: "POST", body: { name } });
}

export function updateAttribute(id: string, name: string): Promise<Attribute> {
  return apiFetch<Attribute>(`/api/v1/admin/attributes/${id}`, { method: "PATCH", body: { name } });
}

export function deleteAttribute(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/attributes/${id}`, { method: "DELETE" });
}

export function addAttributeValue(attributeId: string, value: string): Promise<AttributeValue> {
  return apiFetch<AttributeValue>(`/api/v1/admin/attributes/${attributeId}/values`, {
    method: "POST",
    body: { value },
  });
}

export function deleteAttributeValue(attributeId: string, valueId: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/attributes/${attributeId}/values/${valueId}`, { method: "DELETE" });
}
