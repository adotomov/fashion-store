import { useEffect, useState } from "react";

import { FormField } from "../ui/FormField";
import { Input } from "../ui/Input";
import { Textarea } from "../ui/Textarea";
import { type Language, listLanguages } from "../../lib/api/languages";
import { type TranslatableEntityType, getTranslations, setTranslations } from "../../lib/api/translations";

type TranslationFieldsProps = {
  entityType: TranslatableEntityType;
  // Undefined while the entity hasn't been created yet — translations can
  // only be set once the entity (and its ID) exists.
  entityId?: string;
  // `multiline` renders the field as a resizable Textarea (for longer copy
  // like descriptions) instead of a single-line Input.
  fields: { key: string; label: string; multiline?: boolean }[];
};

// Renders one input per (non-default language) x field, only once the
// store has more than one language — a single-language store has nothing
// to translate. Saves on blur, same idiom as the rest of the admin forms.
export function TranslationFields({ entityType, entityId, fields }: TranslationFieldsProps) {
  const [languages, setLanguages] = useState<Language[]>([]);
  const [values, setValues] = useState<Record<string, Record<string, string>>>({});

  useEffect(() => {
    listLanguages()
      .then((langs) => setLanguages(langs.filter((l) => !l.is_default)))
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!entityId) return;
    let cancelled = false;
    Promise.all(
      languages.map((lang) =>
        getTranslations(entityType, entityId, lang.code).then((fieldValues) => [lang.code, fieldValues] as const),
      ),
    )
      .then((entries) => {
        if (cancelled) return;
        setValues(Object.fromEntries(entries));
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [entityType, entityId, languages]);

  if (languages.length === 0) return null;

  if (!entityId) {
    return (
      <p className="text-xs text-stone-500">Save this item first, then come back to add translations.</p>
    );
  }

  function handleInput(locale: string, field: string, value: string) {
    setValues((prev) => ({ ...prev, [locale]: { ...prev[locale], [field]: value } }));
  }

  async function handleBlur(locale: string, field: string, value: string) {
    await setTranslations(entityType, entityId!, locale, { [field]: value });
  }

  return (
    <div className="flex flex-col gap-4 rounded-sm border border-stone-200 bg-stone-50 p-4">
      {languages.map((lang) => (
        <div key={lang.code} className="flex flex-col gap-3">
          <p className="text-xs font-medium uppercase tracking-wide text-stone-500">{lang.name}</p>
          {fields.map((field) => (
            <FormField key={field.key} label={field.label} htmlFor={`translation-${entityType}-${lang.code}-${field.key}`}>
              {field.multiline ? (
                <Textarea
                  id={`translation-${entityType}-${lang.code}-${field.key}`}
                  value={values[lang.code]?.[field.key] ?? ""}
                  onChange={(e) => handleInput(lang.code, field.key, e.target.value)}
                  onBlur={(e) => handleBlur(lang.code, field.key, e.target.value)}
                />
              ) : (
                <Input
                  id={`translation-${entityType}-${lang.code}-${field.key}`}
                  value={values[lang.code]?.[field.key] ?? ""}
                  onChange={(e) => handleInput(lang.code, field.key, e.target.value)}
                  onBlur={(e) => handleBlur(lang.code, field.key, e.target.value)}
                />
              )}
            </FormField>
          ))}
        </div>
      ))}
    </div>
  );
}
