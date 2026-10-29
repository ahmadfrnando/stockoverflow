import { useState } from "react";
import { NewOrderPage } from "./pages/NewOrderPage";
import { OrderDetailPage } from "./pages/OrderDetailPage";
import { ProductsPage } from "./pages/ProductsPage";

type View =
  | { page: "products" }
  | { page: "new-order" }
  | { page: "order"; id: number };

export default function App() {
  const [view, setView] = useState<View>({ page: "products" });

  return (
    <>
      <nav className="nav">
        <div className="container nav-inner">
          <strong>StockFlow</strong>
          <button
            onClick={() => setView({ page: "products" })}
            aria-current={view.page === "products" ? "page" : undefined}
          >
            Products
          </button>
          <button
            onClick={() => setView({ page: "new-order" })}
            aria-current={view.page === "new-order" ? "page" : undefined}
          >
            New order
          </button>
        </div>
      </nav>

      <main className="container">
        {view.page === "products" && <ProductsPage />}
        {view.page === "new-order" && (
          <NewOrderPage
            onCreated={(order) => setView({ page: "order", id: order.id })}
          />
        )}
        {view.page === "order" && <OrderDetailPage id={view.id} />}
      </main>
    </>
  );
}
