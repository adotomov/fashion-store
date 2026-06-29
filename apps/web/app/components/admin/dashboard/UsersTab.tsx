import { useEffect, useState } from "react";

import { BarList } from "./BarList";
import { DailyChart } from "./DailyChart";
import { Card } from "../../ui/Card";
import { Eyebrow, Text } from "../../ui/Text";
import { type UserStats, getUserStats } from "../../../lib/api/admin-users";
import { COUNTRIES } from "../../../lib/data/countries";

const roleLabels: Record<string, string> = {
  user: "Customer",
  admin: "Admin",
};

function countryName(code: string): string {
  return COUNTRIES.find((c) => c.code === code)?.name ?? code;
}

export function UsersTab() {
  const [stats, setStats] = useState<UserStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getUserStats()
      .then(setStats)
      .catch(() => setError("Could not load user stats."));
  }, []);

  if (error) {
    return (
      <Text size="sm" tone="danger">
        {error}
      </Text>
    );
  }

  if (!stats) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-4">
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            Total users
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.total_users}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            New, last 24h
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.new_24h}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            New, last 7d
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.new_7d}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            New, last 30d
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.new_30d}
          </Text>
        </Card>
      </div>

      <section>
        <Eyebrow>Registrations per day (last 30 days)</Eyebrow>
        <Card className="mt-3 p-6">
          <DailyChart points={stats.daily_registrations.map((d) => ({ date: d.date, value: d.count }))} />
        </Card>
      </section>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <section>
          <Eyebrow>By role</Eyebrow>
          <Card className="mt-3 p-6">
            <BarList items={stats.role_breakdown.map((b) => ({ label: roleLabels[b.label] ?? b.label, count: b.count }))} />
          </Card>
        </section>

        <section>
          <Eyebrow>By country</Eyebrow>
          <Text size="xs" tone="muted" className="mt-1">
            Based on each customer's default address.
          </Text>
          <Card className="mt-3 p-6">
            <BarList items={stats.by_country.map((b) => ({ label: countryName(b.label), count: b.count }))} />
          </Card>
        </section>
      </div>
    </div>
  );
}
