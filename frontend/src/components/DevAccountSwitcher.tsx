import { DEV_ACCOUNTS } from '@chorus/shared'

type Props = {
  onSelect: (a: { email: string; password: string; username: string }) => void
}

// Build-time gate: Vite replaces `import.meta.env.DEV` at bundle time and
// minifiers dead-code-eliminate this entire component (and the `DEV_ACCOUNTS`
// import) from production `dist/` — grep for alice.dev must be 0.
export default function DevAccountSwitcher({ onSelect }: Props) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if (!(import.meta as any).env?.DEV) return null

  return (
    <div className="mb-4 rounded-xl border border-amber-200 bg-amber-50/80 px-3 py-2.5 flex flex-col gap-2">
      <div className="flex items-center gap-1.5">
        <span className="text-[11px] font-bold tracking-widest text-amber-700">DEV ONLY</span>
        <span className="text-xs text-amber-700/70">Test accounts — not in production build</span>
      </div>
      <div className="flex flex-col gap-1.5">
        {DEV_ACCOUNTS.map((a) => (
          <button
            key={a.email}
            type="button"
            onClick={() => onSelect({ email: a.email, username: a.username, password: a.password })}
            className="flex items-center justify-between rounded-lg bg-white border border-amber-200 px-3 py-2 text-left hover:bg-amber-100/60 transition-colors"
          >
            <span className="flex flex-col">
              <span className="text-sm font-medium text-on-surface">{a.label}</span>
              <span className="text-xs text-on-surface-variant">{a.email}</span>
            </span>
            <span className="text-xs font-medium text-primary ml-2 shrink-0">Fill →</span>
          </button>
        ))}
      </div>
      <p className="text-[11px] text-amber-700/60">Password for all: ChorusDev123! — seeded via `go run ./cmd/server --seed-dev`</p>
    </div>
  )
}
