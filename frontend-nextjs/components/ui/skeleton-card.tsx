export function SkeletonCard() {
  return (
    <div className="border rounded-lg p-6 space-y-4">
      <div className="skeleton h-6 w-24" />
      <div className="skeleton h-10 w-32" />
      <div className="skeleton h-4 w-40" />
    </div>
  )
}

