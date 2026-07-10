import { type RouteConfig, index, layout, prefix, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("shop", "routes/shop.tsx"),
  route("shop/:slug", "routes/product-detail.tsx"),
  route("cart", "routes/cart.tsx"),
  route("wishlist", "routes/wishlist.tsx"),
  route("checkout", "routes/checkout.tsx"),
  route("login", "routes/login.tsx"),
  route("about", "routes/about.tsx"),
  ...prefix("legal", [
    route("terms", "routes/legal/terms.tsx"),
    route("privacy", "routes/legal/privacy.tsx"),
  ]),
  ...prefix("help", [
    route("faq", "routes/help/faq.tsx"),
    route("shipping", "routes/help/shipping.tsx"),
  ]),
  ...prefix("account", [
    layout("routes/account/layout.tsx", [
      index("routes/account/personal-info.tsx"),
      route("addresses", "routes/account/addresses.tsx"),
      route("orders", "routes/account/orders.tsx"),
      route("payment-methods", "routes/account/payment-methods.tsx"),
    ]),
  ]),
  route("style-guide", "routes/style-guide.tsx"),
  ...prefix("admin", [
    layout("routes/admin/layout.tsx", [
      index("routes/admin/dashboard.tsx"),
      route("settings", "routes/admin/settings.tsx"),
      route("translations", "routes/admin/translations.tsx"),
      route("orders", "routes/admin/orders.tsx"),
      route("users", "routes/admin/users.tsx"),
      route("catalog", "routes/admin/catalog.tsx"),
      route("products/:id", "routes/admin/product-detail.tsx"),
      route("inventory", "routes/admin/inventory.tsx"),
      route("logistics", "routes/admin/logistics.tsx"),
      route("invoices", "routes/admin/invoices.tsx"),
      route("promotions", "routes/admin/promotions.tsx"),
      route("appearance", "routes/admin/appearance.tsx"),
      route("home", "routes/admin/home.tsx"),
    ]),
  ]),
] satisfies RouteConfig;
