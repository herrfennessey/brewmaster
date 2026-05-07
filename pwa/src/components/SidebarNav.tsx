import type { ReactElement } from 'react'
import { NavLink } from 'react-router-dom'

type IconComp = () => ReactElement

function ScanIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 7V5a2 2 0 0 1 2-2h2" />
      <path d="M17 3h2a2 2 0 0 1 2 2v2" />
      <path d="M21 17v2a2 2 0 0 1-2 2h-2" />
      <path d="M7 21H5a2 2 0 0 1-2-2v-2" />
      <line x1="6" y1="12" x2="18" y2="12" />
    </svg>
  )
}

function CoffeeIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M5 9h11v7a4 4 0 0 1-4 4H9a4 4 0 0 1-4-4V9z" />
      <path d="M16 11h2a3 3 0 0 1 0 6h-2" />
      <path d="M9 5q-1 -1.5 0 -3" />
      <path d="M13 5q-1 -1.5 0 -3" />
    </svg>
  )
}

function ShotDoctorIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M9 12l2 2 4-4" />
    </svg>
  )
}

const items: { to: string; label: string; end?: boolean; Icon: IconComp }[] = [
  { to: '/', label: 'Scan', end: true, Icon: ScanIcon },
  { to: '/coffees', label: 'My coffees', Icon: CoffeeIcon },
  { to: '/shot-doctor', label: 'Shot Doctor', Icon: ShotDoctorIcon },
]

export default function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="sidebar-nav" aria-label="Main">
      <ul className="sidebar-nav__list">
        {items.map(({ to, label, end, Icon }) => (
          <li key={to}>
            <NavLink
              to={to}
              end={end}
              className={({ isActive }) => `sidebar-nav__item${isActive ? ' sidebar-nav__item--active' : ''}`}
              onClick={onNavigate}
            >
              <span className="sidebar-nav__icon"><Icon /></span>
              <span className="sidebar-nav__text">{label}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}
