import { useEffect } from 'react'

export default function Modal({ open, onClose, title, children, size = 'md' }) {
  useEffect(() => {
    const onKey = (e) => e.key === 'Escape' && onClose()
    if (open) document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  // Lock scroll while open
  useEffect(() => {
    if (open) document.body.style.overflow = 'hidden'
    else document.body.style.overflow = ''
    return () => { document.body.style.overflow = '' }
  }, [open])

  if (!open) return null

  const maxW = { sm: 'sm:max-w-md', md: 'sm:max-w-lg', lg: 'sm:max-w-2xl', xl: 'sm:max-w-4xl' }

  return (
    <div className="fixed inset-0 z-50 flex flex-col justify-end sm:justify-center sm:items-center sm:p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Panel
          Mobile:  full-width, slides up from bottom, rounded top corners, max 90vh
          Desktop: centered card, rounded all corners, max 90vh  */}
      <div
        className={[
          'relative w-full bg-white flex flex-col',
          'rounded-t-2xl sm:rounded-xl',
          'max-h-[92vh] sm:max-h-[90vh]',
          'shadow-2xl',
          maxW[size] ?? maxW.md,
        ].join(' ')}
      >
        {/* Drag handle — mobile only */}
        <div className="sm:hidden flex justify-center pt-3 pb-1 flex-shrink-0">
          <div className="h-1 w-10 bg-gray-300 rounded-full" />
        </div>

        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100 flex-shrink-0">
          <h2 className="text-base font-semibold text-gray-900 leading-tight">{title}</h2>
          <button
            onClick={onClose}
            className="p-1.5 -mr-1 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
            aria-label="Close"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Scrollable body */}
        <div className="overflow-y-auto overscroll-contain flex-1 p-5">
          {children}
        </div>
      </div>
    </div>
  )
}
