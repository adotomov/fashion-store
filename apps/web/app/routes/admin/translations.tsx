import { useEffect, useMemo, useState } from "react";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";

import { Badge } from "../../components/ui/Badge";
import { Card } from "../../components/ui/Card";
import { Input } from "../../components/ui/Input";
import { Select } from "../../components/ui/Select";
import { Eyebrow, Text } from "../../components/ui/Text";
import { type Language, listLanguages } from "../../lib/api/languages";
import { type UIString, listAllUiStrings, upsertUiString } from "../../lib/api/ui-strings";

export const handle = { title: "Translations" };

export default function AdminTranslations() {
  const { isReadOnly } = useAdminPermissions();
  const [languages, setLanguages] = useState<Language[]>([]);
  const [strings, setStrings] = useState<UIString[]>([]);
  const [locale, setLocale] = useState("en");
  const [search, setSearch] = useState("");
  const [savingKey, setSavingKey] = useState<string | null>(null);

  async function refresh() {
    setStrings(await listAllUiStrings());
  }

  useEffect(() => {
    listLanguages()
      .then((langs) => {
        setLanguages(langs);
        const firstNonDefault = langs.find((l) => !l.is_default);
        if (firstNonDefault) setLocale(firstNonDefault.code);
      })
      .catch(() => {});
    refresh();
  }, []);

  const englishByKey = useMemo(() => {
    const map = new Map<string, string>();
    for (const s of strings) if (s.locale === "en") map.set(s.key, s.value);
    return map;
  }, [strings]);

  const valueByKey = useMemo(() => {
    const map = new Map<string, string>();
    for (const s of strings) if (s.locale === locale) map.set(s.key, s.value);
    return map;
  }, [strings, locale]);

  const keys = useMemo(() => {
    const all = Array.from(englishByKey.keys()).sort();
    if (!search.trim()) return all;
    const q = search.trim().toLowerCase();
    return all.filter((key) => key.toLowerCase().includes(q) || (englishByKey.get(key) ?? "").toLowerCase().includes(q));
  }, [englishByKey, search]);

  async function handleChange(key: string, value: string) {
    setSavingKey(key);
    try {
      await upsertUiString(key, locale, value);
      setStrings((prev) => {
        const next = prev.filter((s) => !(s.key === key && s.locale === locale));
        next.push({ key, locale, value });
        return next;
      });
    } finally {
      setSavingKey(null);
    }
  }

  const translatableLanguages = languages.filter((l) => !l.is_default);

  if (translatableLanguages.length === 0) {
    return (
      <Text size="sm" tone="muted">
        Add a second language in Store Settings → Store Language before translating static page text.
      </Text>
    );
  }

  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <div>
        <Eyebrow>Translations</Eyebrow>
        <Text size="sm" tone="muted" className="mt-2">
          Translate the static text used across the storefront (buttons, labels, headings). The admin dashboard itself
          always stays in English.
        </Text>
      </div>

      <div className="flex flex-wrap items-end gap-4">
        <div className="w-48">
          <Text size="xs" tone="muted" className="mb-1.5">
            Language
          </Text>
          <Select value={locale} onChange={(e) => setLocale(e.target.value)}>
            {translatableLanguages.map((lang) => (
              <option key={lang.code} value={lang.code}>
                {lang.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="flex-1">
          <Text size="xs" tone="muted" className="mb-1.5">
            Search
          </Text>
          <Input placeholder="Search keys or English text…" value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
      </div>

      <Card className="overflow-hidden p-0">
        <table className="w-full text-sm">
          <thead className="border-b border-stone-200 bg-stone-50 text-left text-xs uppercase tracking-wide text-stone-500">
            <tr>
              <th className="px-4 py-3 font-medium">Key</th>
              <th className="px-4 py-3 font-medium">English</th>
              <th className="px-4 py-3 font-medium">Translation</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((key) => {
              const value = valueByKey.get(key) ?? "";
              return (
                <tr key={key} className="border-b border-stone-100 last:border-0">
                  <td className="px-4 py-3 align-top">
                    <Text size="xs" tone="muted" className="font-mono">
                      {key}
                    </Text>
                  </td>
                  <td className="px-4 py-3 align-top">
                    <Text size="sm">{englishByKey.get(key)}</Text>
                  </td>
                  <td className="px-4 py-3 align-top">
                    <div className="flex items-center gap-2">
                      <Input
                        defaultValue={value}
                        placeholder={!value ? "Not translated yet" : undefined}
                        disabled={isReadOnly}
                        onBlur={(e) => {
                          if (!isReadOnly && e.target.value !== value) handleChange(key, e.target.value);
                        }}
                      />
                      {!value && <Badge variant="accent">Missing</Badge>}
                      {savingKey === key && (
                        <Text size="xs" tone="muted">
                          Saving…
                        </Text>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
