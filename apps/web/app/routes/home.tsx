import type { Route } from "./+types/home";
import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { Hero } from "../components/ecommerce/Hero";
import { NewArrivals } from "../components/ecommerce/NewArrivals";
import { SaleHighlights } from "../components/ecommerce/SaleHighlights";
import { ShopByCategory } from "../components/ecommerce/ShopByCategory";
import { ShopByCollection } from "../components/ecommerce/ShopByCollection";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Maison — Fashion Store" },
    { name: "description", content: "Clothing, jewelry, bags, and accessories." },
  ];
}

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <Hero />
        <NewArrivals />
        <ShopByCategory />
        <ShopByCollection />
        <SaleHighlights />
      </main>
      <Footer />
    </div>
  );
}
