import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listCoffeesAPI } from '../services/api'
import { useAuth } from '../services/auth-context'
import { metaJoin, relativeTime } from '../services/format'
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
      .then(r => {
        if (cancelled) return
        const open = r.coffees
          .filter(c => c.open_bag)
          .sort((a, b) => (b.open_bag!.opened_at).localeCompare(a.open_bag!.opened_at))
          .slice(0, MAX_RAIL_ITEMS)
        setCoffees(open)
      })
      .catch(() => { if (!cancelled) setCoffees([]) })
    return () => { cancelled = true }
  }, [ready])

  if (!coffees || coffees.length === 0) return null

  return (
    <section className="coffees-rail">
      <div className="coffees-rail__head">
        <span className="section-tag">Open bags</span>
        <Link to="/coffees" className="coffees-rail__view-all">View all coffees →</Link>
      </div>
      <div className="coffees-rail__viewport" role="list">
        {coffees.map(c => {
          const bag = c.open_bag!
          const meta = cardMeta(c)
          const opened = relativeTime(bag.opened_at)
          const roast = bag.roast_date ? `Roasted ${bag.roast_date}` : 'No roast date'
          const refillNote = c.bag_count > 1 ? `Bag ${c.bag_count}` : null
          return (
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
              {meta && <span className="coffees-rail__meta">{meta}</span>}
              <span className="coffees-rail__bag-line">
                <span>{roast}</span>
                <span className="coffees-rail__sep">·</span>
                <span>Opened {opened}</span>
              </span>
              <div className="coffees-rail__foot">
                <Stars rating={c.rating} />
                {refillNote && <span className="coffees-rail__refill">{refillNote}</span>}
              </div>
            </Link>
          )
        })}
      </div>
    </section>
  )
}
