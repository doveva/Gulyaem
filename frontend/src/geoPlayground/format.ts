export function formatNumber(value: number): string {
  return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 }).format(value)
}

export function formatInteger(value: number): string {
  return new Intl.NumberFormat('ru-RU').format(value)
}

export function formatDistance(meters: number): string {
  return meters >= 1000 ? `${formatNumber(meters / 1000)} км` : `${formatNumber(meters)} м`
}
