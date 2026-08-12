export function isGeoPlaygroundPath(pathname: string): boolean {
  return pathname === '/debug/geo' || pathname === '/debug/geo/'
}

export function isProductMapPath(pathname: string): boolean {
  return pathname === '/map' || pathname === '/map/'
}
