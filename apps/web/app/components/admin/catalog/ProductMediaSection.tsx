import { useEffect, useRef, useState } from "react";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { Button } from "../../ui/Button";
import { Icon } from "../../ui/Icon";
import { Text } from "../../ui/Text";
import {
  type ProductMedia,
  deleteMedia,
  loadMediaBlobUrl,
  updateMedia,
  uploadProductMedia,
} from "../../../lib/api/products";

type ProductMediaSectionProps = {
  productId: string;
  media: ProductMedia[];
  onChange: () => void;
};

export function ProductMediaSection({ productId, media, onChange }: ProductMediaSectionProps) {
  const { isReadOnly } = useAdminPermissions();
  const [previews, setPreviews] = useState<Record<string, string>>({});
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    const urls: string[] = [];

    async function loadAll() {
      for (const item of media) {
        if (previews[item.id]) continue;
        try {
          const url = await loadMediaBlobUrl(productId, item.id);
          urls.push(url);
          if (!cancelled) {
            setPreviews((prev) => ({ ...prev, [item.id]: url }));
          }
        } catch {
          // skip broken thumbnails rather than blocking the whole section
        }
      }
    }
    loadAll();

    return () => {
      cancelled = true;
      urls.forEach((url) => URL.revokeObjectURL(url));
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [media]);

  async function handleFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setIsUploading(true);
    setError(null);
    try {
      await uploadProductMedia(productId, file, media.length, "");
      onChange();
    } catch {
      setError("Could not upload image.");
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function handleMove(item: ProductMedia, direction: -1 | 1) {
    const sorted = [...media].sort((a, b) => a.position - b.position);
    const index = sorted.findIndex((m) => m.id === item.id);
    const swapWith = sorted[index + direction];
    if (!swapWith) return;

    try {
      await Promise.all([
        updateMedia(productId, item.id, { position: swapWith.position }),
        updateMedia(productId, swapWith.id, { position: item.position }),
      ]);
      onChange();
    } catch {
      setError("Could not reorder images.");
    }
  }

  async function handleAltTextChange(item: ProductMedia, altText: string) {
    try {
      await updateMedia(productId, item.id, { alt_text: altText });
    } catch {
      setError("Could not update alt text.");
    }
  }

  async function handleDelete(item: ProductMedia) {
    if (!window.confirm("Remove this image?")) return;
    try {
      await deleteMedia(productId, item.id);
      onChange();
    } catch {
      setError("Could not remove image.");
    }
  }

  const sorted = [...media].sort((a, b) => a.position - b.position);

  return (
    <div className="flex flex-col gap-4">
      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {sorted.map((item, index) => (
          <div key={item.id} className="flex flex-col gap-2">
            <div className="relative aspect-square overflow-hidden rounded-sm border border-stone-200 bg-stone-50">
              {previews[item.id] ? (
                <img src={previews[item.id]} alt={item.alt_text} className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-stone-400">
                  <Icon name="catalog" size={20} />
                </div>
              )}
              {index === 0 && (
                <span className="absolute left-1.5 top-1.5 rounded-full bg-stone-900/80 px-2 py-0.5 text-[10px] font-medium text-white">
                  Spotlight
                </span>
              )}
            </div>
            <input
              type="text"
              placeholder="Alt text"
              defaultValue={item.alt_text}
              onBlur={(e) => { if (!isReadOnly) handleAltTextChange(item, e.target.value); }}
              disabled={isReadOnly}
              className="h-8 rounded-sm border border-stone-300 px-2 text-xs disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400"
            />
            <div className="flex items-center justify-between">
              <div className="flex gap-1">
                <button
                  type="button"
                  aria-label="Move earlier"
                  disabled={index === 0 || isReadOnly}
                  onClick={() => handleMove(item, -1)}
                  className="rounded-sm p-1 text-stone-500 hover:bg-stone-100 disabled:opacity-30"
                >
                  <Icon name="chevronLeft" size={14} />
                </button>
                <button
                  type="button"
                  aria-label="Move later"
                  disabled={index === sorted.length - 1 || isReadOnly}
                  onClick={() => handleMove(item, 1)}
                  className="rounded-sm p-1 text-stone-500 hover:bg-stone-100 disabled:opacity-30"
                >
                  <Icon name="chevronRight" size={14} />
                </button>
              </div>
              <button
                type="button"
                aria-label="Remove image"
                disabled={isReadOnly}
                onClick={() => handleDelete(item)}
                className="rounded-sm p-1 text-danger-600 hover:bg-danger-50 disabled:pointer-events-none disabled:opacity-30"
              >
                <Icon name="trash" size={14} />
              </button>
            </div>
          </div>
        ))}

        {!isReadOnly && (
          <label className="flex aspect-square cursor-pointer flex-col items-center justify-center gap-2 rounded-sm border border-dashed border-stone-300 text-stone-500 hover:border-stone-400 hover:text-stone-700">
            <Icon name="plus" size={20} />
            <Text size="xs">{isUploading ? "Uploading…" : "Add image"}</Text>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              onChange={handleFileSelected}
              disabled={isUploading}
              className="hidden"
            />
          </label>
        )}
      </div>
    </div>
  );
}
