// Placeholder card shown while the available-pets query is in flight.
// Mimics the shape of a real PetCard so the grid doesn't reflow when data arrives.
export function PetCardSkeleton() {
  return (
    <article className="card card--skeleton" aria-hidden="true">
      <div className="card__image skeleton" />
      <div className="card__body">
        <div className="skeleton skeleton--line skeleton--line-lg" />
        <div className="skeleton skeleton--line skeleton--line-sm" />
        <div className="skeleton skeleton--line" />
        <div className="skeleton skeleton--line skeleton--line-md" />
      </div>
      <div className="card__actions">
        <div className="skeleton skeleton--btn" />
        <div className="skeleton skeleton--btn" />
      </div>
    </article>
  );
}

export function PetCardSkeletonGrid({ count = 6 }: { count?: number }) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <PetCardSkeleton key={i} />
      ))}
    </>
  );
}
