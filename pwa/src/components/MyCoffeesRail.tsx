import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listCoffeesAPI } from '../services/api'
import { useAuth } from '../services/auth-context'
import { metaJoin } from '../services/format'
import type { CoffeeSummary } from '../types'

const MAX_RAIL_ITEMS = 8

function Stars({ rating }: { rating?: number }) {
  if (!rating) return null
  return (
    <span className="coffees-rail__stars" aria-label={`${rating} of 5`}>
      <span className="coffees-rail__stars-filled">{'★'.repeat(rating)}</span>
      <span className="coffees-rail__stars-empty">{'★'.repeat(5 - rating)}</span>
    </span>
  )
}

function cardTitle(c: CoffeeSummary): string {
  const card = c.bean_card
  if (card.producer) return card.producer
  return metaJoin([card.origin_region, card.origin_country], ', ') || 'Untitled coffee'
}

function cardMeta(c: CoffeeSummary): string {
  return metaJoin([c.bean_card.varietal, c.bean_card.process, c.bean_card.roast_level])
}

export default function MyCoffeesRail() {
  const { ready } = useAuth()
  const [coffees, setCoffees] = useState<CoffeeSummary[] | null>(null)

  useEffect(() => {
    if (!ready) return
    let cancelled = false
    listCoffeesAPI()
      .then(r => { if (!cancelled) setCoffees(r.coffees.slice(0, MAX_RAIL_ITEMS)) })
      .catch(() => { if (!cancelled) setCoffees([]) })
    return () => { cancelled = true }
  }, [ready])

  if (!coffees || coffees.length === 0) return null

  return (
    <section className="coffees-rail">
      <div className="coffees-rail__head">
        <span className="section-tag">My coffees</span>
        <Link to="/coffees" className="coffees-rail__view-all">View all →</Link>
      </div>
      <div className="coffees-rail__viewport" role="list">
        {coffees.map(c => (
          <Link
            key={c.coffee_id}
            to={`/coffees/${c.coffee_id}`}
            className="coffees-rail__card"
            role="listitem"
          >
            {c.bean_card.roaster_name && (
              <span className="coffees-rail__roaster">{c.bean_card.roaster_name}</span>
            )}
            <span className="coffees-rail__title">{cardTitle(c)}</span>
            {cardMeta(c) && <span className="coffees-rail__meta">{cardMeta(c)}</span>}
            <Stars rating={c.rating} />
          </Link>
        ))}
      </div>
    </section>
  )
}
