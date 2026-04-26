export default function Pagination({ total, limit, offset, onPageChange }) {
  const totalPages = Math.ceil(total / limit)
  const currentPage = Math.floor(offset / limit) + 1

  if (totalPages <= 1) return null

  const pages = []
  const delta = 1
  for (let i = 1; i <= totalPages; i++) {
    if (
      i === 1 ||
      i === totalPages ||
      (i >= currentPage - delta && i <= currentPage + delta)
    ) {
      pages.push(i)
    } else if (pages[pages.length - 1] !== '...') {
      pages.push('...')
    }
  }

  return (
    <div className="flex items-center justify-between px-4 py-3 border-t border-gray-100 gap-2">
      {/* Count */}
      <p className="text-xs sm:text-sm text-gray-500 flex-shrink-0">
        <span className="hidden sm:inline">Showing </span>
        {offset + 1}–{Math.min(offset + limit, total)}
        <span className="hidden sm:inline"> of {total}</span>
        <span className="sm:hidden"> / {total}</span>
      </p>

      {/* Controls */}
      <div className="flex items-center gap-1 flex-shrink-0">
        <button
          onClick={() => onPageChange(offset - limit)}
          disabled={currentPage === 1}
          className="px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm rounded-lg border border-gray-200 disabled:opacity-40 hover:bg-gray-50 active:bg-gray-100 transition-colors font-medium"
        >
          ← <span className="hidden sm:inline">Prev</span>
        </button>

        {/* Page numbers — hidden on xs, visible on sm+ */}
        <div className="hidden sm:flex gap-1">
          {pages.map((p, i) =>
            p === '...' ? (
              <span key={`e-${i}`} className="px-2 py-1.5 text-xs text-gray-400">…</span>
            ) : (
              <button
                key={p}
                onClick={() => onPageChange((p - 1) * limit)}
                className={`w-8 h-8 text-xs rounded-lg border transition-colors font-medium ${
                  p === currentPage
                    ? 'bg-brand-600 text-white border-brand-600'
                    : 'border-gray-200 hover:bg-gray-50'
                }`}
              >
                {p}
              </button>
            )
          )}
        </div>

        {/* Current / total on xs */}
        <span className="sm:hidden px-2.5 py-1.5 text-xs text-gray-500 font-medium">
          {currentPage}/{totalPages}
        </span>

        <button
          onClick={() => onPageChange(offset + limit)}
          disabled={currentPage === totalPages}
          className="px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm rounded-lg border border-gray-200 disabled:opacity-40 hover:bg-gray-50 active:bg-gray-100 transition-colors font-medium"
        >
          <span className="hidden sm:inline">Next</span> →
        </button>
      </div>
    </div>
  )
}
