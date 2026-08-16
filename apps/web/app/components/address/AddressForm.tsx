import { useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { searchComplexes, searchSites, searchStreets, type Site } from "../../lib/api/logistics";
import { isValidPhone } from "../../lib/phone";
import { Combobox, type ComboboxOption } from "../ui/Combobox";
import { FormField } from "../ui/FormField";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

// StructuredAddress is the Speedy-resolved address shared by the profile
// address book and checkout. City, complex (кв./жк.) and street carry Speedy
// location codes (…_id) alongside their display names; house details are free
// text. Bulgaria is the only supported country for now.
export type StructuredAddress = {
  recipient_name: string;
  phone: string;
  country_code: string;
  country_id: number;
  site_id: number;
  city: string;
  post_code: string;
  complex_id: number;
  complex_name: string;
  street_id: number;
  street_name: string;
  street_no: string;
  block_no: string;
  entrance_no: string;
  floor_no: string;
  apartment_no: string;
};

export const emptyStructuredAddress: StructuredAddress = {
  recipient_name: "",
  phone: "",
  country_code: "BG",
  country_id: 100,
  site_id: 0,
  city: "",
  post_code: "",
  complex_id: 0,
  complex_name: "",
  street_id: 0,
  street_name: "",
  street_no: "",
  block_no: "",
  entrance_no: "",
  floor_no: "",
  apartment_no: "",
};

// isStructuredAddressComplete mirrors the backend's required fields: a resolved
// city, complex and street. Used to gate submit buttons.
export function isStructuredAddressComplete(a: StructuredAddress): boolean {
  return a.site_id > 0 && a.complex_id > 0 && a.street_id > 0;
}

type AddressFormProps = {
  value: StructuredAddress;
  onChange: (next: StructuredAddress) => void;
  /** Show recipient name + phone (profile). Checkout collects those separately. */
  showRecipient?: boolean;
  idPrefix?: string;
};

export function AddressForm({ value, onChange, showRecipient = false, idPrefix = "addr" }: AddressFormProps) {
  const { t } = useLanguage();
  const [phoneError, setPhoneError] = useState<string | null>(null);

  function patch(fields: Partial<StructuredAddress>) {
    onChange({ ...value, ...fields });
  }

  const siteSublabel = (s: Site) => [s.region, s.post_code].filter(Boolean).join(" · ");

  return (
    <div className="flex flex-col gap-4">
      {showRecipient && (
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField label={t("address.recipient_name", "Recipient name")} htmlFor={`${idPrefix}-recipient`}>
            <Input
              id={`${idPrefix}-recipient`}
              value={value.recipient_name}
              onChange={(e) => patch({ recipient_name: e.target.value })}
            />
          </FormField>
          <FormField label={t("address.phone", "Phone")} htmlFor={`${idPrefix}-phone`} error={phoneError ?? undefined}>
            <Input
              id={`${idPrefix}-phone`}
              type="tel"
              invalid={!!phoneError}
              value={value.phone}
              onChange={(e) => {
                patch({ phone: e.target.value });
                if (phoneError) setPhoneError(null);
              }}
              onBlur={(e) =>
                setPhoneError(
                  e.target.value.trim() && !isValidPhone(e.target.value)
                    ? t("address.phone_invalid", "Enter a valid phone number, e.g. 0888 123 456 or +359 88 812 3456.")
                    : null,
                )
              }
            />
          </FormField>
        </div>
      )}

      <FormField label={t("address.country", "Country")} htmlFor={`${idPrefix}-country`}>
        <Select id={`${idPrefix}-country`} value="BG" disabled>
          <option value="BG">{t("address.country_bg", "Bulgaria")}</option>
        </Select>
      </FormField>

      <FormField label={t("address.city", "City")} htmlFor={`${idPrefix}-city`}>
        <Combobox
          id={`${idPrefix}-city`}
          value={value.city}
          placeholder={t("address.city_placeholder", "Start typing a city…")}
          emptyText={t("address.no_cities", "No cities found")}
          onValueChange={(text) =>
            // Editing the city invalidates the resolved city and everything
            // scoped to it (complex, street).
            patch({
              city: text,
              site_id: 0,
              post_code: "",
              complex_id: 0,
              complex_name: "",
              street_id: 0,
              street_name: "",
            })
          }
          onSearch={async (q) =>
            (await searchSites(q)).map<ComboboxOption>((s) => ({ id: s.id, label: s.name, sublabel: siteSublabel(s) }))
          }
          onSelect={(opt) => {
            void searchSites(opt.label).then((sites) => {
              const site = sites.find((s) => s.id === opt.id);
              patch({
                site_id: Number(opt.id),
                city: opt.label,
                post_code: site?.post_code ?? "",
                complex_id: 0,
                complex_name: "",
                street_id: 0,
                street_name: "",
              });
            });
          }}
        />
      </FormField>

      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          label={t("address.complex", "Neighbourhood (кв./жк.)")}
          htmlFor={`${idPrefix}-complex`}
          hint={value.site_id === 0 ? t("address.pick_city_first", "Pick a city first") : undefined}
        >
          <Combobox
            id={`${idPrefix}-complex`}
            value={value.complex_name}
            disabled={value.site_id === 0}
            placeholder={t("address.complex_placeholder", "Start typing…")}
            emptyText={t("address.no_complexes", "No neighbourhoods found")}
            onValueChange={(text) => patch({ complex_name: text, complex_id: 0 })}
            onSearch={async (q) =>
              (await searchComplexes(value.site_id, q)).map<ComboboxOption>((c) => ({
                id: c.id,
                label: c.name,
                sublabel: c.type,
              }))
            }
            onSelect={(opt) => patch({ complex_id: Number(opt.id), complex_name: opt.label })}
          />
        </FormField>

        <FormField
          label={t("address.street", "Street")}
          htmlFor={`${idPrefix}-street`}
          hint={value.site_id === 0 ? t("address.pick_city_first", "Pick a city first") : undefined}
        >
          <Combobox
            id={`${idPrefix}-street`}
            value={value.street_name}
            disabled={value.site_id === 0}
            placeholder={t("address.street_placeholder", "Start typing…")}
            emptyText={t("address.no_streets", "No streets found")}
            onValueChange={(text) => patch({ street_name: text, street_id: 0 })}
            onSearch={async (q) =>
              (await searchStreets(value.site_id, q)).map<ComboboxOption>((s) => ({
                id: s.id,
                label: s.name,
                sublabel: s.type,
              }))
            }
            onSelect={(opt) => patch({ street_id: Number(opt.id), street_name: opt.label })}
          />
        </FormField>
      </div>

      <div className="grid gap-4 sm:grid-cols-4">
        <FormField label={t("address.street_no", "Street No.")} htmlFor={`${idPrefix}-street-no`}>
          <Input
            id={`${idPrefix}-street-no`}
            value={value.street_no}
            onChange={(e) => patch({ street_no: e.target.value })}
          />
        </FormField>
        <FormField label={t("address.block_no", "Block")} htmlFor={`${idPrefix}-block`}>
          <Input id={`${idPrefix}-block`} value={value.block_no} onChange={(e) => patch({ block_no: e.target.value })} />
        </FormField>
        <FormField label={t("address.floor_no", "Floor")} htmlFor={`${idPrefix}-floor`}>
          <Input id={`${idPrefix}-floor`} value={value.floor_no} onChange={(e) => patch({ floor_no: e.target.value })} />
        </FormField>
        <FormField label={t("address.apartment_no", "Apartment")} htmlFor={`${idPrefix}-apartment`}>
          <Input
            id={`${idPrefix}-apartment`}
            value={value.apartment_no}
            onChange={(e) => patch({ apartment_no: e.target.value })}
          />
        </FormField>
      </div>
    </div>
  );
}
