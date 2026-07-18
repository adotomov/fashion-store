import { useState } from "react";

import { Breadcrumbs } from "../components/ecommerce/Breadcrumbs";
import { type FilterGroup, FilterPanel } from "../components/ecommerce/FilterPanel";
import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { ProductCard } from "../components/ecommerce/ProductCard";
import { ProductImageGallery } from "../components/ecommerce/ProductImageGallery";
import { ProductInfo } from "../components/ecommerce/ProductInfo";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Checkbox } from "../components/ui/Checkbox";
import { FormField } from "../components/ui/FormField";
import { Input } from "../components/ui/Input";
import { Price } from "../components/ui/Price";
import { QuantityStepper } from "../components/ui/QuantityStepper";
import { Rating } from "../components/ui/Rating";
import { Select } from "../components/ui/Select";
import { Textarea } from "../components/ui/Textarea";
import { Eyebrow, Heading, Text } from "../components/ui/Text";

const swatches = [
  { name: "stone-50", className: "bg-stone-50" },
  { name: "stone-100", className: "bg-stone-100" },
  { name: "stone-200", className: "bg-stone-200" },
  { name: "stone-300 (beige)", className: "bg-stone-300" },
  { name: "stone-400", className: "bg-stone-400" },
  { name: "stone-500", className: "bg-stone-500" },
  { name: "stone-600", className: "bg-stone-600" },
  { name: "stone-700", className: "bg-stone-700" },
  { name: "stone-800", className: "bg-stone-800" },
  { name: "stone-900 (charcoal)", className: "bg-stone-900" },
  { name: "clay-500", className: "bg-clay-500" },
  { name: "sage-500", className: "bg-sage-500" },
  { name: "danger-500", className: "bg-danger-500" },
];

const filterGroups: FilterGroup[] = [
  {
    id: "category",
    label: "Category",
    type: "checkbox",
    options: [
      { id: "clothing", label: "Clothing", count: 128 },
      { id: "jewelry", label: "Jewelry", count: 54 },
      { id: "bags", label: "Bags", count: 32 },
      { id: "accessories", label: "Accessories", count: 76 },
    ],
  },
  {
    id: "color",
    label: "Color",
    type: "color",
    options: [
      { name: "Beige", hex: "#DDD0C8" },
      { name: "Charcoal", hex: "#323232" },
      { name: "Clay", hex: "#B2543C" },
      { name: "Ivory", hex: "#FAF8F6" },
    ],
  },
];

export default function StyleGuide() {
  const [filters, setFilters] = useState<Record<string, string[]>>({});
  const [wishlisted, setWishlisted] = useState<Record<string, boolean>>({});

  function toggleFilter(groupId: string, optionId: string) {
    setFilters((prev) => {
      const current = prev[groupId] ?? [];
      const next = current.includes(optionId)
        ? current.filter((id) => id !== optionId)
        : [...current, optionId];
      return { ...prev, [groupId]: next };
    });
  }

  return (
    <div className="flex min-h-screen flex-col">
      <Header />

      <main className="mx-auto flex max-w-7xl flex-col gap-16 px-4 py-12 sm:px-6 lg:px-8">
        <section>
          <Breadcrumbs items={[{ label: "Home", href: "/" }, { label: "Style Guide" }]} />
          <Heading as="h1" size="xl" className="mt-4">
            Brand Style Guide
          </Heading>
          <Text tone="muted" className="mt-2 max-w-2xl">
            Reference page for the theme and component library. Not part of the storefront navigation.
          </Text>
        </section>

        <section>
          <Eyebrow>Color Palette</Eyebrow>
          <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4 md:grid-cols-6">
            {swatches.map((s) => (
              <div key={s.name}>
                <div className={`aspect-square w-full rounded-sm border border-black/5 ${s.className}`} />
                <Text size="xs" tone="muted" className="mt-1.5">
                  {s.name}
                </Text>
              </div>
            ))}
          </div>
        </section>

        <section>
          <Eyebrow>Typography</Eyebrow>
          <div className="mt-4 flex flex-col gap-3">
            <Heading as="h1" size="xl">Heading XL — Display Serif</Heading>
            <Heading as="h2" size="lg">Heading LG</Heading>
            <Heading as="h3" size="md">Heading MD</Heading>
            <Heading as="h4" size="sm">Heading SM</Heading>
            <Text size="lg">Body large — used for product descriptions.</Text>
            <Text size="md">Body medium — the default body copy size.</Text>
            <Text size="sm" tone="muted">Body small, muted — secondary/meta text.</Text>
            <Eyebrow>Eyebrow label</Eyebrow>
          </div>
        </section>

        <section>
          <Eyebrow>Buttons</Eyebrow>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <Button variant="primary">Primary</Button>
            <Button variant="secondary">Secondary</Button>
            <Button variant="outline">Outline</Button>
            <Button variant="ghost">Ghost</Button>
            <Button variant="accent">Accent</Button>
            <Button variant="danger">Danger</Button>
            <Button variant="primary" disabled>Disabled</Button>
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <Button variant="primary" size="sm">Small</Button>
            <Button variant="primary" size="md">Medium</Button>
            <Button variant="primary" size="lg">Large</Button>
          </div>
        </section>

        <section>
          <Eyebrow>Forms</Eyebrow>
          <Card className="mt-4 max-w-md p-6">
            <div className="flex flex-col gap-4">
              <FormField label="Full name" htmlFor="name">
                <Input id="name" placeholder="Jane Doe" />
              </FormField>
              <FormField label="Email" htmlFor="email" error="Please enter a valid email address">
                <Input id="email" type="email" placeholder="jane@example.com" invalid />
              </FormField>
              <FormField label="Country" htmlFor="country">
                <Select id="country" defaultValue="">
                  <option value="" disabled>Select a country</option>
                  <option value="bg">Bulgaria</option>
                  <option value="de">Germany</option>
                  <option value="us">United States</option>
                </Select>
              </FormField>
              <FormField label="Notes" htmlFor="notes" hint="Optional delivery instructions">
                <Textarea id="notes" placeholder="Leave at the front desk" />
              </FormField>
              <Checkbox id="newsletter" label="Subscribe to our newsletter" />
              <Button variant="primary">Submit</Button>
            </div>
          </Card>
        </section>

        <section>
          <Eyebrow>Badges, Price &amp; Rating</Eyebrow>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <Badge variant="neutral">New</Badge>
            <Badge variant="brand">Limited</Badge>
            <Badge variant="accent">-20%</Badge>
            <Badge variant="success">In Stock</Badge>
            <Badge variant="danger">Sold Out</Badge>
          </div>
          <div className="mt-4 flex flex-wrap items-center gap-6">
            <Price price={{ amount: 12999, currency: "EUR" }} size="lg" />
            <Price price={{ amount: 8999, currency: "EUR" }} compareAtPrice={{ amount: 12999, currency: "EUR" }} size="lg" />
            <Rating value={4} count={128} />
          </div>
          <div className="mt-4">
            <QuantityStepperDemo />
          </div>
        </section>

        <section>
          <Eyebrow>Breadcrumbs</Eyebrow>
          <Breadcrumbs
            className="mt-4"
            items={[
              { label: "Home", href: "/" },
              { label: "Clothing", href: "/catalog/clothing" },
              { label: "Dresses", href: "/catalog/clothing/dresses" },
              { label: "Silk Wrap Dress" },
            ]}
          />
        </section>

        <section className="grid grid-cols-1 gap-10 lg:grid-cols-[280px_1fr]">
          <div>
            <Eyebrow>Dynamic Filter</Eyebrow>
            <FilterPanel
              className="mt-4"
              groups={filterGroups}
              selected={filters}
              onToggle={toggleFilter}
              onClear={() => setFilters({})}
            />
          </div>

          <div>
            <Eyebrow>Product Card Grid</Eyebrow>
            <div className="mt-4 grid grid-cols-2 gap-6 sm:grid-cols-3">
              {[1, 2, 3].map((i) => (
                <ProductCard
                  key={i}
                  href="/style-guide"
                  image={{ src: `https://picsum.photos/seed/product-${i}/600/800`, alt: "Sample product" }}
                  title="Silk Wrap Dress in Beige"
                  price={{ amount: 8999, currency: "EUR" }}
                  compareAtPrice={i === 1 ? { amount: 12999, currency: "EUR" } : undefined}
                  badge={i === 1 ? "Sale" : undefined}
                  isWishlisted={wishlisted[i] ?? false}
                  onToggleWishlist={() => setWishlisted((prev) => ({ ...prev, [i]: !prev[i] }))}
                />
              ))}
            </div>
          </div>
        </section>

        <section>
          <Eyebrow>Product Page</Eyebrow>
          <div className="mt-4 grid grid-cols-1 gap-10 lg:grid-cols-2">
            <ProductImageGallery
              main={{ src: "https://picsum.photos/seed/product-main/900/900", alt: "Silk Wrap Dress" }}
              thumbnails={[
                { src: "https://picsum.photos/seed/product-2/900/900", alt: "Detail view" },
                { src: "https://picsum.photos/seed/product-3/900/900", alt: "Back view" },
                { src: "https://picsum.photos/seed/product-4/900/900", alt: "Fabric detail" },
              ]}
            />
            <ProductInfo
              title="Silk Wrap Dress"
              description="A timeless wrap dress cut from 100% mulberry silk, finished with hand-rolled hems. Designed to move with you, from morning meetings to evening dinners."
              tags={["New", "Best Seller"]}
              price={{ amount: 8999, currency: "EUR" }}
              compareAtPrice={{ amount: 12999, currency: "EUR" }}
              variants={[
                { label: "XS" },
                { label: "S" },
                { label: "M" },
                { label: "L", available: false },
                { label: "XL" },
              ]}
              colors={[
                { name: "Beige", hex: "#DDD0C8" },
                { name: "Charcoal", hex: "#323232" },
                { name: "Clay", hex: "#B2543C" },
              ]}
            />
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}

function QuantityStepperDemo() {
  const [quantity, setQuantity] = useState(1);
  return <QuantityStepper quantity={quantity} onChange={setQuantity} />;
}
