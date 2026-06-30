import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { type WishlistItem, addToWishlist, listWishlist, removeFromWishlist } from "../../lib/api/wishlist";
import { useAuth } from "../auth/AuthContext";

type WishlistContextValue = {
  items: WishlistItem[];
  isLoading: boolean;
  count: number;
  isWishlisted: (productId: string) => boolean;
  toggle: (productId: string) => Promise<void>;
  refresh: () => Promise<void>;
};

const WishlistContext = createContext<WishlistContextValue | undefined>(undefined);

export function WishlistProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  const [items, setItems] = useState<WishlistItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  async function refresh() {
    if (!isAuthenticated) {
      setItems([]);
      setIsLoading(false);
      return;
    }
    try {
      setItems(await listWishlist());
    } catch {
      // leave previous state on a transient fetch failure
    } finally {
      setIsLoading(false);
    }
  }

  // Wishlisting requires auth — re-sync whenever sign-in state changes
  // (login picks up the user's saved items, logout clears the local list).
  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated]);

  function isWishlisted(productId: string): boolean {
    return items.some((item) => item.product_id === productId);
  }

  async function toggle(productId: string) {
    if (isWishlisted(productId)) {
      await removeFromWishlist(productId);
      setItems((prev) => prev.filter((item) => item.product_id !== productId));
    } else {
      const item = await addToWishlist(productId);
      setItems((prev) => [item, ...prev]);
    }
  }

  return (
    <WishlistContext.Provider value={{ items, isLoading, count: items.length, isWishlisted, toggle, refresh }}>
      {children}
    </WishlistContext.Provider>
  );
}

export function useWishlist(): WishlistContextValue {
  const ctx = useContext(WishlistContext);
  if (!ctx) {
    throw new Error("useWishlist must be used within a WishlistProvider");
  }
  return ctx;
}
