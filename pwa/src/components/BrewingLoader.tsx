interface BrewingLoaderProps {
  message: string
}

export default function BrewingLoader({ message }: BrewingLoaderProps) {
  return (
    <div className="brewing-loader" role="status" aria-live="polite">
      <div className="brewing-loader__scene">
        <div className="brewing-loader__steam" aria-hidden="true">
          <div className="brewing-loader__steam-line" />
          <div className="brewing-loader__steam-line" />
          <div className="brewing-loader__steam-line" />
        </div>
        <div className="brewing-loader__stream" aria-hidden="true" />
        <div className="brewing-loader__cup" aria-hidden="true">
          <div className="brewing-loader__crema" />
        </div>
      </div>
      <p className="brewing-loader__message">{message}</p>
    </div>
  )
}
