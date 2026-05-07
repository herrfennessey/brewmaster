import { Link } from 'react-router-dom'

export default function ShotDoctor() {
  return (
    <div className="screen shot-doctor-screen">
      <span className="section-tag">Shot Doctor</span>
      <h1 className="shot-doctor__heading">How did the shot taste?</h1>
      <p className="shot-doctor__copy">
        Coming soon. Pull a shot, log how it tasted (sour, bitter, weak, balanced…)
        and Brewmaster will suggest a deterministic tweak — temperature up,
        ratio in, time out — for the next round.
      </p>
      <p className="shot-doctor__copy">
        While we're building it, you can already save coffees and dial in
        starting parameters. Pick the bag you brewed today to start there.
      </p>
      <Link to="/coffees" className="action-btn action-btn--primary shot-doctor__cta">
        Pick a coffee →
      </Link>
    </div>
  )
}
