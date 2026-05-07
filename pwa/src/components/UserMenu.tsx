import { useEffect, useRef, useState } from 'react'
import { useAuth } from '../services/auth-context'

function CaretIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="3 4.5 6 7.5 9 4.5" />
    </svg>
  )
}

function firstName(user: { displayName: string | null; email: string | null }): string {
  if (user.displayName) return user.displayName.split(' ')[0]
  if (user.email) return user.email.split('@')[0]
  return 'You'
}

function initial(name: string): string {
  return name.charAt(0).toUpperCase()
}

export default function UserMenu() {
  const { user, isAnonymous, signIn, signOut } = useAuth()
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    function onEsc(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onEsc)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onEsc)
    }
  }, [open])

  async function handleSignIn() {
    setBusy(true)
    try {
      await signIn()
    } catch (err) {
      console.error('sign in failed', err)
    } finally {
      setBusy(false)
    }
  }

  if (!user || isAnonymous) {
    return (
      <button
        type="button"
        className="user-menu__signin"
        onClick={handleSignIn}
        disabled={busy}
      >
        {busy ? 'Signing in…' : 'Sign in'}
      </button>
    )
  }

  const name = firstName(user)
  return (
    <div className="user-menu" ref={ref}>
      <button
        type="button"
        className="user-menu__chip"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen(o => !o)}
      >
        <span className="user-menu__avatar">{initial(name)}</span>
        <span className="user-menu__name">{name}</span>
        <CaretIcon />
      </button>
      {open && (
        <div className="user-menu__dropdown" role="menu">
          {user.email && <div className="user-menu__email">{user.email}</div>}
          <button
            type="button"
            className="user-menu__action"
            role="menuitem"
            onClick={() => { setOpen(false); void signOut() }}
          >
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}
