import { useEffect, useState } from 'react'
import { Outlet } from 'react-router-dom'
import SidebarNav from './SidebarNav'
import UserMenu from './UserMenu'

const MOBILE_QUERY = '(max-width: 899px)'

function CupIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 32 30" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <rect x="4" y="10" width="18" height="14" rx="2.5" stroke="currentColor" strokeWidth="1.5"/>
      <path d="M22 13.5 Q29 13.5 29 18 Q29 22.5 22 22.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" fill="none"/>
      <path d="M2.5 24 Q11.5 28 24 24" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
      <path d="M10 7 Q9 4.5 11 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" opacity="0.55"/>
      <path d="M16 7 Q15 4.5 17 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" opacity="0.55"/>
    </svg>
  )
}

function HamburgerIcon({ open }: { open: boolean }) {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true">
      {open ? (
        <>
          <line x1="6" y1="6" x2="18" y2="18" />
          <line x1="18" y1="6" x2="6" y2="18" />
        </>
      ) : (
        <>
          <line x1="4" y1="7" x2="20" y2="7" />
          <line x1="4" y1="12" x2="20" y2="12" />
          <line x1="4" y1="17" x2="20" y2="17" />
        </>
      )}
    </svg>
  )
}

export default function AppShell() {
  const [isMobile, setIsMobile] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)

  // Track viewport width via matchMedia. Closing the drawer on the same
  // listener avoids leaving an open overlay if the user resizes past the
  // breakpoint while the drawer is open.
  useEffect(() => {
    const mq = window.matchMedia(MOBILE_QUERY)
    const sync = () => {
      setIsMobile(mq.matches)
      if (!mq.matches) setDrawerOpen(false)
    }
    sync()
    mq.addEventListener('change', sync)
    return () => mq.removeEventListener('change', sync)
  }, [])

  // Lock body scroll while the mobile drawer is open so the underlying content
  // doesn't scroll behind the overlay.
  useEffect(() => {
    if (!(isMobile && drawerOpen)) return
    const original = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = original }
  }, [isMobile, drawerOpen])

  return (
    <div className="app-shell">
      <header className="app-shell__topbar">
        {isMobile && (
          <button
            type="button"
            className="app-shell__hamburger"
            aria-label={drawerOpen ? 'Close navigation' : 'Open navigation'}
            aria-expanded={drawerOpen}
            onClick={() => setDrawerOpen(o => !o)}
          >
            <HamburgerIcon open={drawerOpen} />
          </button>
        )}
        <div className="app-shell__brand">
          <CupIcon className="app-shell__brand-icon" />
          <span className="app-shell__brand-name">Brewmaster</span>
        </div>
        <UserMenu />
      </header>

      <div className="app-shell__body">
        {isMobile && (
          <div
            className={`app-shell__backdrop${drawerOpen ? ' app-shell__backdrop--visible' : ''}`}
            onClick={() => setDrawerOpen(false)}
            aria-hidden="true"
          />
        )}
        <aside
          className={
            'app-shell__sidebar' +
            (isMobile ? ' app-shell__sidebar--mobile' : '') +
            (isMobile && drawerOpen ? ' app-shell__sidebar--open' : '')
          }
          aria-label="Sidebar"
        >
          <SidebarNav onNavigate={isMobile ? () => setDrawerOpen(false) : undefined} />
        </aside>
        <main className="app-shell__content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
