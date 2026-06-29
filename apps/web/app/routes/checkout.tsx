import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { Heading } from "../components/ui/Text";
import { CheckoutFlow } from "../features/checkout/CheckoutFlow";

export const handle = { title: "Checkout" };

export default function Checkout() {
  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1 bg-stone-50">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
          <Heading as="h1" size="lg">
            Checkout
          </Heading>
          <div className="mt-8">
            <CheckoutFlow />
          </div>
        </div>
      </main>
      <Footer />
    </div>
  );
}
