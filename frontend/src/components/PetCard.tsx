import type { Pet } from "../types";

interface Props {
  pet: Pet;
  inCart: boolean;
  busy: boolean;
  onBuyNow: () => void;
  onAddToCart: () => void;
  onRemoveFromCart: () => void;
}

const FALLBACK_IMG =
  "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 4 3'><rect width='4' height='3' fill='%23eee'/></svg>";

export function PetCard({ pet, inCart, busy, onBuyNow, onAddToCart, onRemoveFromCart }: Props) {
  return (
    <article className="card">
      <div className="card__image-wrap">
        <img
          className="card__image"
          src={pet.pictureUrl}
          alt={pet.name}
          loading="lazy"
          onError={(e) => {
            (e.target as HTMLImageElement).src = FALLBACK_IMG;
          }}
        />
      </div>
      <div className="card__body">
        <h3 className="card__title">{pet.name}</h3>
        <div className="card__meta">
          <span className="species-tag">{pet.species}</span>{" "}
          · {pet.age} {pet.age === 1 ? "year" : "years"} old
        </div>
        <p className="card__desc">{pet.description}</p>
      </div>
      <div className="card__actions">
        <button className="btn btn--primary" disabled={busy} onClick={onBuyNow}>
          Buy now
        </button>
        {inCart ? (
          <button className="btn btn--secondary" disabled={busy} onClick={onRemoveFromCart}>
            Remove from cart
          </button>
        ) : (
          <button className="btn btn--secondary" disabled={busy} onClick={onAddToCart}>
            Add to cart
          </button>
        )}
      </div>
    </article>
  );
}
