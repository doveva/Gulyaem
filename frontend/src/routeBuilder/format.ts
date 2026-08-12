export function formatDistance(meters: number): string {
  return meters < 1000 ? `${Math.round(meters)} м` : `${(meters / 1000).toFixed(1).replace('.', ',')} км`
}

export function formatDuration(seconds: number): string {
  const minutes = Math.max(1, Math.round(seconds / 60))
  if (minutes < 60) return `${minutes} мин`
  const hours = Math.floor(minutes / 60)
  const remainder = minutes % 60
  return remainder === 0 ? `${hours} ч` : `${hours} ч ${remainder} мин`
}
