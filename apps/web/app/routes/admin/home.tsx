import { useEffect, useRef, useState } from "react";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Tabs } from "../../components/ui/Tabs";
import { Text } from "../../components/ui/Text";
import {
  type EditorialBannerSettings,
  type HeroSettings,
  type SaveEditorialBannerInput,
  type SaveHeroSettingsInput,
  deleteEditorialBannerImage,
  deleteHeroBackground,
  getEditorialBanner,
  getHeroSettings,
  saveEditorialBanner,
  saveHeroSettings,
  uploadEditorialBannerImage,
  uploadHeroBackground,
} from "../../lib/api/admin-appearance";
import {
  type HomeSectionConfig,
  type SectionCategoryGroup,
  getAdminSectionProductIDs,
  getSectionCategoryGroups,
  listAdminHomeSections,
  saveHomeSection,
  setSectionCategoryGroups,
  setSectionProducts,
} from "../../lib/api/admin-home-sections";
import { type Category, listCategories } from "../../lib/api/categories";
import { type Product, listProducts } from "../../lib/api/products";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";

export const handle = { title: "Home Page" };

const TABS = [
  { id: "hero", label: "Hero" },
  { id: "sections", label: "Sections" },
];

export default function AdminHomePage() {
  const [activeTab, setActiveTab] = useState("hero");

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <Tabs tabs={TABS} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === "hero" && <HeroTab />}
        {activeTab === "sections" && <SectionsTab />}
      </Tabs>
    </div>
  );
}

// ─── Hero Tab ─────────────────────────────────────────────────────────────────

type FormState = {
  eyebrow: string;
  heading: string;
  subtext: string;
  cta_primary_label: string;
  cta_primary_url: string;
  cta_secondary_label: string;
  cta_secondary_url: string;
};

function settingsToForm(s: HeroSettings): FormState {
  return {
    eyebrow: s.eyebrow,
    heading: s.heading,
    subtext: s.subtext,
    cta_primary_label: s.cta_primary_label,
    cta_primary_url: s.cta_primary_url,
    cta_secondary_label: s.cta_secondary_label ?? "",
    cta_secondary_url: s.cta_secondary_url ?? "",
  };
}

const emptyForm: FormState = {
  eyebrow: "",
  heading: "",
  subtext: "",
  cta_primary_label: "",
  cta_primary_url: "",
  cta_secondary_label: "",
  cta_secondary_url: "",
};

function HeroTab() {
  const { isReadOnly } = useAdminPermissions();
  const [form, setForm] = useState<FormState>(emptyForm);
  const [backgroundImageUrl, setBackgroundImageUrl] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploadingBg, setUploadingBg] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getHeroSettings()
      .then((s) => {
        setForm(settingsToForm(s));
        setBackgroundImageUrl(s.background_image_url);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function set(field: keyof FormState, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSuccess(false);
    setError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setSuccess(false);
    setError(null);
    try {
      const input: SaveHeroSettingsInput = {
        eyebrow: form.eyebrow,
        heading: form.heading,
        subtext: form.subtext,
        cta_primary_label: form.cta_primary_label,
        cta_primary_url: form.cta_primary_url,
        cta_secondary_label: form.cta_secondary_label || undefined,
        cta_secondary_url: form.cta_secondary_url || undefined,
      };
      const saved = await saveHeroSettings(input);
      setForm(settingsToForm(saved));
      setBackgroundImageUrl(saved.background_image_url);
      setSuccess(true);
    } catch {
      setError("Failed to save hero settings.");
    } finally {
      setSaving(false);
    }
  }

  async function handleBackgroundUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploadingBg(true);
    setError(null);
    try {
      const saved = await uploadHeroBackground(file);
      setBackgroundImageUrl(saved.background_image_url);
    } catch {
      setError("Failed to upload background image.");
    } finally {
      setUploadingBg(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function handleDeleteBackground() {
    setUploadingBg(true);
    setError(null);
    try {
      const saved = await deleteHeroBackground();
      setBackgroundImageUrl(saved.background_image_url);
    } catch {
      setError("Failed to remove background image.");
    } finally {
      setUploadingBg(false);
    }
  }

  if (loading) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-6">
      {/* Background image */}
      <Card className="flex flex-col gap-5 p-6">
        <Text size="sm" className="font-semibold text-stone-900">
          Background Image
        </Text>
        <Text size="xs" tone="muted">
          Uploaded image replaces the default gradient. Recommended: 1920×1080px or wider, JPG/WebP.
        </Text>

        {backgroundImageUrl ? (
          <div className="flex flex-col gap-3">
            <div className="overflow-hidden rounded-sm border border-stone-200">
              <img
                src={resolveImageUrl(backgroundImageUrl)}
                alt="Hero background"
                className="h-40 w-full object-cover"
              />
            </div>
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploadingBg || isReadOnly}
              >
                Replace
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => void handleDeleteBackground()}
                disabled={uploadingBg || isReadOnly}
                className="text-danger-600 hover:bg-danger-50"
              >
                {uploadingBg ? "Removing…" : "Remove"}
              </Button>
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={uploadingBg || isReadOnly}
            >
              {uploadingBg ? "Uploading…" : "Upload Image"}
            </Button>
            <Text size="xs" tone="muted">
              No image — gradient is used.
            </Text>
          </div>
        )}

        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(e) => void handleBackgroundUpload(e)}
        />
      </Card>

      {/* Text content */}
      <Card className="flex flex-col gap-5 p-6">
        <Text size="sm" className="font-semibold text-stone-900">
          Content
        </Text>

        <FormField label="Eyebrow Text">
          <Input
            value={form.eyebrow}
            onChange={(e) => set("eyebrow", e.target.value)}
            placeholder="e.g. New Season"
          />
        </FormField>

        <FormField label="Heading">
          <Input
            value={form.heading}
            onChange={(e) => set("heading", e.target.value)}
            placeholder="e.g. Quietly considered style, for every day."
          />
        </FormField>

        <FormField label="Subtext">
          <textarea
            value={form.subtext}
            onChange={(e) => set("subtext", e.target.value)}
            rows={3}
            placeholder="Supporting text shown below the heading."
            className="w-full rounded-sm border border-stone-300 bg-white px-3.5 py-2.5 text-sm text-stone-900 placeholder:text-stone-400 transition-colors focus:border-stone-900 focus:outline-none disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400 resize-none"
          />
        </FormField>

        <div className="flex flex-col gap-2">
          <Text size="sm" className="font-medium text-stone-700">
            Primary CTA Button
          </Text>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Button Label">
              <Input
                value={form.cta_primary_label}
                onChange={(e) => set("cta_primary_label", e.target.value)}
                placeholder="e.g. Shop All Items"
              />
            </FormField>
            <FormField label="Button URL">
              <Input
                value={form.cta_primary_url}
                onChange={(e) => set("cta_primary_url", e.target.value)}
                placeholder="e.g. /shop"
              />
            </FormField>
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Text size="sm" className="font-medium text-stone-700">
            Secondary CTA Button
          </Text>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Button Label" hint="Leave blank to hide">
              <Input
                value={form.cta_secondary_label}
                onChange={(e) => set("cta_secondary_label", e.target.value)}
                placeholder="e.g. View the Sale"
              />
            </FormField>
            <FormField label="Button URL" hint="Leave blank to hide">
              <Input
                value={form.cta_secondary_url}
                onChange={(e) => set("cta_secondary_url", e.target.value)}
                placeholder="e.g. /shop?sale=true"
              />
            </FormField>
          </div>
        </div>
      </Card>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}
      {success && (
        <Text size="sm" className="text-sage-600">
          Hero settings saved.
        </Text>
      )}

      <div className="flex justify-end">
        <Button type="submit" disabled={saving || isReadOnly}>
          {saving ? "Saving…" : "Save Changes"}
        </Button>
      </div>
    </form>
  );
}

// ─── Sections Tab ─────────────────────────────────────────────────────────────

const SECTION_LABELS: Record<string, string> = {
  spotlights: "Spotlights",
  recommended: "Recommended by Us",
  on_sale: "What's on Sale",
  best_in_category: "Best in its Category",
};

const CURATED_SECTIONS = new Set(["spotlights", "recommended"]);

function SectionsTab() {
  const [sections, setSections] = useState<HomeSectionConfig[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  async function refresh() {
    try {
      setSections(await listAdminHomeSections());
    } catch {
      setLoadError("Could not load home section settings.");
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  if (loadError) {
    return (
      <Text size="sm" tone="danger">
        {loadError}
      </Text>
    );
  }

  if (!sections) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <EditorialBannerCard />
      {sections.map((section) => (
        <SectionCard key={section.id} section={section} onSaved={refresh} />
      ))}
    </div>
  );
}

// ─── Editorial banner card ────────────────────────────────────────────────────

// The editorial ("Shop the Look") banner is a singleton with its own settings
// (image + copy + CTA), separate from the home_sections rows above, so it has a
// dedicated card here rather than going through SectionCard.
type BannerFormState = {
  enabled: boolean;
  eyebrow: string;
  heading: string;
  subtext: string;
  cta_label: string;
  cta_url: string;
};

function bannerToForm(b: EditorialBannerSettings): BannerFormState {
  return {
    enabled: b.enabled,
    eyebrow: b.eyebrow,
    heading: b.heading,
    subtext: b.subtext,
    cta_label: b.cta_label,
    cta_url: b.cta_url,
  };
}

function EditorialBannerCard() {
  const { isReadOnly } = useAdminPermissions();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [form, setForm] = useState<BannerFormState | null>(null);
  const [imageUrl, setImageUrl] = useState<string | undefined>(undefined);
  const [isSaving, setIsSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    getEditorialBanner()
      .then((b) => {
        setForm(bannerToForm(b));
        setImageUrl(b.image_url);
      })
      .catch(() => {});
  }, []);

  function set<K extends keyof BannerFormState>(field: K, value: BannerFormState[K]) {
    setForm((prev) => (prev ? { ...prev, [field]: value } : prev));
    setSavedAt(null);
    setSaveError(null);
  }

  async function handleSave(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!form) return;
    setIsSaving(true);
    setSaveError(null);
    try {
      const input: SaveEditorialBannerInput = { ...form };
      const saved = await saveEditorialBanner(input);
      setForm(bannerToForm(saved));
      setImageUrl(saved.image_url);
      setSavedAt(Date.now());
    } catch {
      setSaveError("Could not save. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setSaveError(null);
    try {
      const saved = await uploadEditorialBannerImage(file);
      setForm(bannerToForm(saved));
      setImageUrl(saved.image_url);
    } catch {
      setSaveError("Failed to upload image.");
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function handleRemoveImage() {
    setUploading(true);
    setSaveError(null);
    try {
      const saved = await deleteEditorialBannerImage();
      setForm(bannerToForm(saved));
      setImageUrl(saved.image_url);
    } catch {
      setSaveError("Failed to remove image.");
    } finally {
      setUploading(false);
    }
  }

  if (!form) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <section>
      <div className="mb-3 flex items-center gap-3">
        <h2 className="text-base font-semibold text-stone-900">Editorial Banner</h2>
        {form.enabled ? (
          <Badge variant="success">Enabled</Badge>
        ) : (
          <Badge variant="neutral">Disabled</Badge>
        )}
      </div>

      <Card className="p-6">
        <form className="flex flex-col gap-5" onSubmit={(e) => void handleSave(e)}>
          <Text size="xs" tone="muted">
            A large "Shop the Look" banner shown mid-home-page, after Shop by Collection.
          </Text>

          <label className="flex items-center gap-3 text-sm text-stone-700">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => set("enabled", e.target.checked)}
              className="h-4 w-4 rounded border-stone-300 text-stone-900 focus:ring-stone-900"
            />
            Show this section on the home page
          </label>

          {/* Banner image */}
          <div className="flex flex-col gap-3">
            <Text size="sm" className="font-medium text-stone-700">
              Banner Image
            </Text>
            <Text size="xs" tone="muted">
              Wide lifestyle image works best (text is overlaid on the left). Recommended: 1600×900px or wider.
            </Text>
            {imageUrl ? (
              <div className="flex flex-col gap-3">
                <div className="overflow-hidden rounded-sm border border-stone-200">
                  <img src={resolveImageUrl(imageUrl)} alt="Editorial banner" className="h-40 w-full object-cover" />
                </div>
                <div className="flex items-center gap-3">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploading || isReadOnly}
                  >
                    Replace
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => void handleRemoveImage()}
                    disabled={uploading || isReadOnly}
                    className="text-danger-600 hover:bg-danger-50"
                  >
                    {uploading ? "Removing…" : "Remove"}
                  </Button>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploading || isReadOnly}
                >
                  {uploading ? "Uploading…" : "Upload Image"}
                </Button>
                <Text size="xs" tone="muted">
                  No image — gradient is used.
                </Text>
              </div>
            )}
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => void handleUpload(e)}
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Eyebrow text" htmlFor="editorial-eyebrow">
              <Input
                id="editorial-eyebrow"
                value={form.eyebrow}
                onChange={(e) => set("eyebrow", e.target.value)}
                placeholder="e.g. The Edit"
              />
            </FormField>
            <FormField label="Heading" htmlFor="editorial-heading">
              <Input
                id="editorial-heading"
                value={form.heading}
                onChange={(e) => set("heading", e.target.value)}
                placeholder="e.g. Shop the look"
              />
            </FormField>
          </div>

          <FormField label="Subtext" htmlFor="editorial-subtext">
            <textarea
              id="editorial-subtext"
              value={form.subtext}
              onChange={(e) => set("subtext", e.target.value)}
              rows={3}
              placeholder="Supporting text shown below the heading."
              className="w-full rounded-sm border border-stone-300 bg-white px-3.5 py-2.5 text-sm text-stone-900 placeholder:text-stone-400 transition-colors focus:border-stone-900 focus:outline-none disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400 resize-none"
            />
          </FormField>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Button Label" htmlFor="editorial-cta-label">
              <Input
                id="editorial-cta-label"
                value={form.cta_label}
                onChange={(e) => set("cta_label", e.target.value)}
                placeholder="e.g. Explore the edit"
              />
            </FormField>
            <FormField label="Button URL" htmlFor="editorial-cta-url">
              <Input
                id="editorial-cta-url"
                value={form.cta_url}
                onChange={(e) => set("cta_url", e.target.value)}
                placeholder="e.g. /shop"
              />
            </FormField>
          </div>

          <div className="flex items-center gap-3">
            <Button type="submit" variant="primary" size="sm" disabled={isSaving || isReadOnly}>
              {isSaving ? "Saving…" : "Save"}
            </Button>
            {saveError && (
              <Text size="xs" tone="danger">
                {saveError}
              </Text>
            )}
            {!saveError && savedAt && (
              <Text size="xs" tone="muted">
                Saved.
              </Text>
            )}
          </div>
        </form>
      </Card>
    </section>
  );
}

// ─── Individual section card ──────────────────────────────────────────────────

function SectionCard({
  section,
  onSaved,
}: {
  section: HomeSectionConfig;
  onSaved: () => void;
}) {
  const { isReadOnly } = useAdminPermissions();
  const [enabled, setEnabled] = useState(section.enabled);
  const [eyebrow, setEyebrow] = useState(section.eyebrow);
  const [heading, setHeading] = useState(section.heading);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    setEnabled(section.enabled);
    setEyebrow(section.eyebrow);
    setHeading(section.heading);
  }, [section.enabled, section.eyebrow, section.heading]);

  async function handleSave(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsSaving(true);
    setSaveError(null);
    try {
      await saveHomeSection(section.id, { enabled, eyebrow, heading });
      setSavedAt(Date.now());
      onSaved();
    } catch {
      setSaveError("Could not save. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <section>
      <div className="mb-3 flex items-center gap-3">
        <h2 className="text-base font-semibold text-stone-900">
          {SECTION_LABELS[section.id] ?? section.id}
        </h2>
        {enabled ? (
          <Badge variant="success">Enabled</Badge>
        ) : (
          <Badge variant="neutral">Disabled</Badge>
        )}
      </div>

      <Card className="p-6">
        <form className="flex flex-col gap-4" onSubmit={(e) => void handleSave(e)}>
          <label className="flex items-center gap-3 text-sm text-stone-700">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4 rounded border-stone-300 text-stone-900 focus:ring-stone-900"
            />
            Show this section on the home page
          </label>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Eyebrow text" htmlFor={`eyebrow-${section.id}`}>
              <Input
                id={`eyebrow-${section.id}`}
                value={eyebrow}
                onChange={(e) => setEyebrow(e.target.value)}
                placeholder="e.g. Curated"
              />
            </FormField>
            <FormField label="Heading" htmlFor={`heading-${section.id}`}>
              <Input
                id={`heading-${section.id}`}
                value={heading}
                onChange={(e) => setHeading(e.target.value)}
                placeholder="e.g. Staff Picks"
              />
            </FormField>
          </div>

          <div className="flex items-center gap-3">
            <Button type="submit" variant="primary" size="sm" disabled={isSaving || isReadOnly}>
              {isSaving ? "Saving…" : "Save"}
            </Button>
            {saveError && (
              <Text size="xs" tone="danger">
                {saveError}
              </Text>
            )}
            {!saveError && savedAt && (
              <Text size="xs" tone="muted">
                Saved.
              </Text>
            )}
          </div>
        </form>

        {CURATED_SECTIONS.has(section.id) && (
          <div className="mt-6 border-t border-stone-100 pt-6">
            <ProductPicker sectionId={section.id} />
          </div>
        )}

        {section.id === "best_in_category" && (
          <div className="mt-6 border-t border-stone-100 pt-6">
            <CategoryGroupPicker sectionId={section.id} />
          </div>
        )}
      </Card>
    </section>
  );
}

// ─── Product picker for curated sections ─────────────────────────────────────

function ProductPicker({ sectionId }: { sectionId: string }) {
  const { isReadOnly } = useAdminPermissions();
  const [allProducts, setAllProducts] = useState<Product[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    listProducts().catch(() => []).then((p) => setAllProducts(p));
    getAdminSectionProductIDs(sectionId).catch(() => []).then((ids) => setSelectedIds(ids));
  }, [sectionId]);

  const selectedProducts = allProducts.filter((p) => selectedIds.includes(p.id));
  const filtered = allProducts.filter(
    (p) =>
      !selectedIds.includes(p.id) &&
      p.name.toLowerCase().includes(search.toLowerCase()),
  );

  function add(id: string) {
    setSelectedIds((prev) => [...prev, id]);
  }

  function remove(id: string) {
    setSelectedIds((prev) => prev.filter((x) => x !== id));
  }

  async function handleSave() {
    setIsSaving(true);
    setSaveError(null);
    try {
      await setSectionProducts(sectionId, selectedIds);
      setSavedAt(Date.now());
    } catch {
      setSaveError("Could not save product list.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Text size="sm" className="font-medium text-stone-700">
        Curated Products
      </Text>

      {selectedProducts.length > 0 && (
        <div className="flex flex-col gap-2">
          {selectedProducts.map((product) => (
            <div
              key={product.id}
              className="flex items-center justify-between gap-3 rounded-sm border border-stone-200 bg-stone-50 px-3 py-2"
            >
              <Text size="sm">{product.name}</Text>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => remove(product.id)}
                disabled={isReadOnly}
                className="text-danger-600 hover:bg-danger-50"
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
      )}

      <div className="flex flex-col gap-2">
        <Input
          placeholder="Search products…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        {search && (
          <div className="flex max-h-60 flex-col gap-1 overflow-y-auto rounded-sm border border-stone-200 bg-white p-1">
            {filtered.length === 0 ? (
              <Text size="sm" tone="muted" className="px-2 py-2">
                No products found.
              </Text>
            ) : (
              filtered.map((product) => (
                <div
                  key={product.id}
                  className="flex items-center justify-between gap-3 rounded-sm px-2 py-1.5 hover:bg-stone-50"
                >
                  <Text size="sm">{product.name}</Text>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => add(product.id)}
                    disabled={isReadOnly}
                  >
                    Add
                  </Button>
                </div>
              ))
            )}
          </div>
        )}
      </div>

      <div className="flex items-center gap-3">
        <Button type="button" variant="primary" size="sm" disabled={isSaving || isReadOnly} onClick={() => void handleSave()}>
          {isSaving ? "Saving…" : "Save Product List"}
        </Button>
        {saveError && (
          <Text size="xs" tone="danger">
            {saveError}
          </Text>
        )}
        {!saveError && savedAt && (
          <Text size="xs" tone="muted">
            Saved.
          </Text>
        )}
      </div>
    </div>
  );
}

// ─── Category-group picker (Best in its category) ─────────────────────────────

const MAX_GROUP_CATEGORIES = 5;
const MAX_PRODUCTS_PER_GROUP = 10;

function CategoryGroupPicker({ sectionId }: { sectionId: string }) {
  const { isReadOnly } = useAdminPermissions();
  const [categories, setCategories] = useState<Category[]>([]);
  const [groups, setGroups] = useState<SectionCategoryGroup[]>([]);
  const [addingCategory, setAddingCategory] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    listCategories().catch(() => []).then(setCategories);
    getSectionCategoryGroups(sectionId).catch(() => []).then(setGroups);
  }, [sectionId]);

  const categoryName = (id: string) => categories.find((c) => c.id === id)?.name ?? id;
  const selectedIds = groups.map((g) => g.category_id);
  const available = categories.filter((c) => !selectedIds.includes(c.id));
  const atCategoryLimit = groups.length >= MAX_GROUP_CATEGORIES;

  function addCategory(categoryId: string) {
    setGroups((prev) => [...prev, { category_id: categoryId, product_ids: [] }]);
    setAddingCategory(false);
  }

  function removeCategory(categoryId: string) {
    setGroups((prev) => prev.filter((g) => g.category_id !== categoryId));
  }

  function setGroupProducts(categoryId: string, productIds: string[]) {
    setGroups((prev) =>
      prev.map((g) => (g.category_id === categoryId ? { ...g, product_ids: productIds } : g)),
    );
  }

  async function handleSave() {
    setIsSaving(true);
    setSaveError(null);
    try {
      await setSectionCategoryGroups(sectionId, groups);
      setSavedAt(Date.now());
    } catch {
      setSaveError("Could not save categories.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <Text size="sm" className="font-medium text-stone-700">
          Categories ({groups.length}/{MAX_GROUP_CATEGORIES})
        </Text>
        <Text size="xs" tone="muted">
          Pick up to {MAX_GROUP_CATEGORIES} categories, then curate up to {MAX_PRODUCTS_PER_GROUP} products in each.
        </Text>
      </div>

      {groups.map((group) => (
        <div key={group.category_id} className="rounded-sm border border-stone-200 p-4">
          <div className="mb-3 flex items-center justify-between gap-3">
            <Text size="sm" className="font-semibold text-stone-900">
              {categoryName(group.category_id)}
            </Text>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => removeCategory(group.category_id)}
              disabled={isReadOnly}
              className="text-danger-600 hover:bg-danger-50"
            >
              Remove category
            </Button>
          </div>
          <CategoryProductPicker
            categoryId={group.category_id}
            selectedIds={group.product_ids}
            onChange={(ids) => setGroupProducts(group.category_id, ids)}
          />
        </div>
      ))}

      {addingCategory ? (
        <div className="flex flex-col gap-1 rounded-sm border border-stone-200 bg-white p-1">
          {available.length === 0 ? (
            <Text size="sm" tone="muted" className="px-2 py-2">
              All categories are already added.
            </Text>
          ) : (
            <div className="max-h-60 overflow-y-auto">
              {available.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => addCategory(c.id)}
                  className="flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm hover:bg-stone-50"
                >
                  {c.name}
                </button>
              ))}
            </div>
          )}
        </div>
      ) : (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setAddingCategory(true)}
          disabled={isReadOnly || atCategoryLimit}
          className="self-start"
        >
          {atCategoryLimit ? `Max ${MAX_GROUP_CATEGORIES} categories` : "Add category"}
        </Button>
      )}

      <div className="flex items-center gap-3">
        <Button type="button" variant="primary" size="sm" disabled={isSaving || isReadOnly} onClick={() => void handleSave()}>
          {isSaving ? "Saving…" : "Save Categories"}
        </Button>
        {saveError && (
          <Text size="xs" tone="danger">
            {saveError}
          </Text>
        )}
        {!saveError && savedAt && (
          <Text size="xs" tone="muted">
            Saved.
          </Text>
        )}
      </div>
    </div>
  );
}

// Per-category product picker: lists the active products in that category and
// lets the admin pick an ordered subset (up to MAX_PRODUCTS_PER_GROUP).
function CategoryProductPicker({
  categoryId,
  selectedIds,
  onChange,
}: {
  categoryId: string;
  selectedIds: string[];
  onChange: (ids: string[]) => void;
}) {
  const { isReadOnly } = useAdminPermissions();
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);
  const [search, setSearch] = useState("");

  useEffect(() => {
    listStorefrontProducts({ categoryIds: [categoryId] })
      .catch(() => [])
      .then(setProducts);
  }, [categoryId]);

  const nameOf = (id: string) => products?.find((p) => p.id === id)?.name ?? id;
  const atProductLimit = selectedIds.length >= MAX_PRODUCTS_PER_GROUP;
  const filtered = (products ?? []).filter(
    (p) => !selectedIds.includes(p.id) && p.name.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="flex flex-col gap-2">
      {selectedIds.length > 0 && (
        <div className="flex flex-col gap-1.5">
          {selectedIds.map((id) => (
            <div
              key={id}
              className="flex items-center justify-between gap-3 rounded-sm border border-stone-200 bg-stone-50 px-3 py-1.5"
            >
              <Text size="sm">{nameOf(id)}</Text>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => onChange(selectedIds.filter((x) => x !== id))}
                disabled={isReadOnly}
                className="text-danger-600 hover:bg-danger-50"
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
      )}

      <Text size="xs" tone="muted">
        {selectedIds.length}/{MAX_PRODUCTS_PER_GROUP} products
        {products !== null && products.length === 0 ? " — no active products in this category" : ""}
      </Text>

      {!atProductLimit && (
        <>
          <Input placeholder="Search products in this category…" value={search} onChange={(e) => setSearch(e.target.value)} />
          {search && (
            <div className="flex max-h-52 flex-col gap-1 overflow-y-auto rounded-sm border border-stone-200 bg-white p-1">
              {filtered.length === 0 ? (
                <Text size="sm" tone="muted" className="px-2 py-2">
                  No products found.
                </Text>
              ) : (
                filtered.map((p) => (
                  <div key={p.id} className="flex items-center justify-between gap-3 rounded-sm px-2 py-1.5 hover:bg-stone-50">
                    <Text size="sm">{p.name}</Text>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => onChange([...selectedIds, p.id])}
                      disabled={isReadOnly}
                    >
                      Add
                    </Button>
                  </div>
                ))
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
