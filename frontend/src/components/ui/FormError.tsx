// Renders a consistent inline error banner, or nothing when there is no error.
// Removes the copy-pasted red alert block every form used to repeat.
export function FormError({ message, className = 'text-sm' }: { message: string | null; className?: string }) {
  if (!message) return null
  return (
    <p className={`rounded-lg border border-red-500/20 bg-red-500/5 px-3 py-2 text-red-400 ${className}`}>
      {message}
    </p>
  )
}