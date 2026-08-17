import { useEffect, useState } from "react";

import type { Route } from "./+types/home";
import { BestInCategory } from "../components/ecommerce/BestInCategory";
import { CategoryBanners } from "../components/ecommerce/CategoryBanners";
import { EditorialBanner } from "../components/ecommerce/EditorialBanner";
import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { Hero } from "../components/ecommerce/Hero";
import { NewArrivals } from "../components/ecommerce/NewArrivals";
import { OnSale } from "../components/ecommerce/OnSale";
import { RecentlyViewed } from "../components/ecommerce/RecentlyViewed";
import { RecommendedByUs } from "../components/ecommerce/RecommendedByUs";
import { ShopByCategory } from "../components/ecommerce/ShopByCategory";
import { ShopByCollection } from "../components/ecommerce/ShopByCollection";
import { Spotlights } from "../components/ecommerce/Spotlights";
import { TrustBar } from "../components/ecommerce/TrustBar";
import { type HomeSectionConfig, getPublicHomeSections } from "../lib/api/admin-home-sections";
import { buildMeta } from "../lib/seo/meta";
import { organizationJsonLd, websiteJsonLd } from "../lib/seo/jsonld";

export function meta({}: Route.MetaArgs) {
  return buildMeta({
    title: "Verani — Clothing, Jewelry, Bags & Accessories",
    titleAsIs: true,
    description:
      "Discover curated fashion at Verani: new-arrival clothing, jewelry, bags, and accessories, with fast delivery across Bulgaria.",
    path: "/",
    jsonLd: [organizationJsonLd(), websiteJsonLd()],
  });
}

export default function Home() {
  const [sections, setSections] = useState<HomeSectionConfig[]>([]);

  useEffect(() => {
    getPublicHomeSections().then(setSections).catch(() => {});
  }, []);

  function getSection(id: string): HomeSectionConfig | undefined {
    return sections.find((s) => s.id === id);
  }

  const spotlightsSection = getSection("spotlights");
  const recommendedSection = getSection("recommended");
  const onSaleSection = getSection("on_sale");
  const bestInCategorySection = getSection("best_in_category");

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <Hero />
        <TrustBar />
        <CategoryBanners />
        <NewArrivals />
        <ShopByCategory />
        <ShopByCollection />
        <EditorialBanner />
        {spotlightsSection && <Spotlights section={spotlightsSection} />}
        {recommendedSection && <RecommendedByUs section={recommendedSection} />}
        {onSaleSection && <OnSale section={onSaleSection} />}
        {bestInCategorySection && <BestInCategory section={bestInCategorySection} />}
        <RecentlyViewed />
      </main>
      <Footer />
    </div>
  );
}
