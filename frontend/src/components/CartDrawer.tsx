import { useCart } from "../cart/CartProvider";

interface Props {
  open: boolean;
  busy: boolean;
  onClose: () => void;
  onCheckout: () => void;
}

export function CartDrawer({ open, busy, onClose, onCheckout }: Props) {
  const { items, remove, count } = useCart();

  return (
    <aside className={`cart ${open ? "cart--open" : ""}`} aria-hidden={!open}>
      <div className="cart__header">
        <strong>Your cart ({count})</strong>
        <button className="btn btn--ghost" onClick={onClose} aria-label="Close cart">×</button>
      </div>
      <ul className="cart__list">
        {items.length === 0 && <li style={{ color: "var(--muted)" }}>Your cart is empty.</li>}
        {items.map((p) => (
          <li className="cart__item" key={p.id}>
            <img src={p.pictureUrl} alt={p.name} />
            <span className="cart__item-name">{p.name}</span>
            <button className="btn btn--ghost" onClick={() => remove(p.id)} disabled={busy}>
              Remove
            </button>
          </li>
        ))}
      </ul>
      <div className="cart__footer">
        <button
          className="btn btn--primary btn--block"
          onClick={onCheckout}
          disabled={busy || items.length === 0}
        >
          {busy ? "Processing…" : `Checkout (${count})`}
        </button>
      </div>
    </aside>
  );
}
