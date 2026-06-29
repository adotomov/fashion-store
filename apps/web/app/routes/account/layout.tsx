import { Outlet, useMatches } from "react-router";

import { AccountSidebar } from "../../components/account/AccountSidebar";
import { Footer } from "../../components/ecommerce/Footer";
import { Header } from "../../components/ecommerce/Header";
import { RequireAuth } from "../../components/RequireAuth";
import { Heading } from "../../components/ui/Text";

export default function AccountLayout() {
  const matches = useMatches();
  const title = (matches.at(-1)?.handle as { title?: string } | undefined)?.title ?? "Personal Info";

  return (
    <RequireAuth>
      <div className="flex min-h-screen flex-col">
        <Header />
        <main className="flex-1 bg-stone-50">
          <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
            <Heading as="h1" size="lg">
              {title}
            </Heading>
            <div className="mt-8 grid grid-cols-1 gap-10 lg:grid-cols-[220px_1fr]">
              <AccountSidebar />
              <div>
                <Outlet />
              </div>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    </RequireAuth>
  );
}
