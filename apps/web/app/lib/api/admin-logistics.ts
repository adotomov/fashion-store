import { apiFetch } from "./client";

export type LogisticsProvider = {
  provider: string;
  name: string;
  enabled: boolean;
  config: Record<string, string>;
  /** True when the API is running this provider against a fake/simulated client (SPEEDY_MODE=fake). */
  dev_mode?: boolean;
  updated_at?: string;
};

export function listProviders(): Promise<LogisticsProvider[]> {
  return apiFetch<LogisticsProvider[]>("/api/v1/admin/logistics/providers");
}

export function saveProvider(
  provider: string,
  input: { enabled: boolean; config: Record<string, string> },
): Promise<LogisticsProvider> {
  return apiFetch<LogisticsProvider>(`/api/v1/admin/logistics/providers/${provider}`, {
    method: "PUT",
    body: input,
  });
}

export type Office = {
  id: string;
  name: string;
  type: string;
};

export function listOffices(provider: string, city: string, type = "APT"): Promise<Office[]> {
  const params = new URLSearchParams({ provider, city, type });
  return apiFetch<Office[]>(`/api/v1/logistics/offices?${params.toString()}`, { auth: false });
}
