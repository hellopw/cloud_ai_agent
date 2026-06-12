const icons = {
  agent:   { viewBox: '0 0 24 24', path: 'M12 2a3 3 0 0 1 3 3v1h2a2 2 0 0 1 2 2v2a3 3 0 0 1 0 6v2a2 2 0 0 1-2 2h-2v1a3 3 0 0 1-6 0v-1H7a2 2 0 0 1-2-2v-2a3 3 0 1 1 0-6v-2a2 2 0 0 1 2-2h2V5a3 3 0 0 1 3-3zm-2 14a2 2 0 1 0 4 0 2 2 0 0 0-4 0z' },
  prompts: { viewBox: '0 0 24 24', path: 'M4 6h16M4 12h16M4 18h10' },
  skills:  { viewBox: '0 0 24 24', path: 'M13 2L3 14h7l-1 8 10-12h-7l1-8z' },
  tools:   { viewBox: '0 0 24 24', path: 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.77 3.77z' },
  memories:{ viewBox: '0 0 24 24', path: 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm0 4a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm-2 8a2 2 0 0 1 4 0v3a2 2 0 0 1-4 0v-3z' },
  templates:{ viewBox: '0 0 24 24', path: 'M3 3h7v7H3V3zm11 0h7v7h-7V3zM3 14h7v7H3v-7zm11 0h7v7h-7v-7z' },
  agents:  { viewBox: '0 0 24 24', path: 'M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0zM12 14c-4.4 0-8 1.8-8 4v2h16v-2c0-2.2-3.6-4-8-4z' },
  teams:   { viewBox: '0 0 24 24', path: 'M12 4a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM5 8a2 2 0 1 0 0 4 2 2 0 0 0 0-4zm14 0a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM5 16c0-1.7 2.7-3 6-3s6 1.3 6 3v1H5v-1zm8 0c0-1.7 1.8-3 4-3s4 1.3 4 3v1h-8v-1zM3 16c0-1.7 1.8-3 4-3s4 1.3 4 3v1H3v-1z' },
  instances:{ viewBox: '0 0 24 24', path: 'M5 3h14a2 2 0 0 1 2 2v4H3V5a2 2 0 0 1 2-2zm-2 8h18v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6zm4 2v2m4-2v2m4-2v2' },
  models:  { viewBox: '0 0 24 24', path: 'M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5' },
  resources:{ viewBox: '0 0 24 24', path: 'M4 7a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7zm4 0v10m8-10v10' },
  palette: { viewBox: '0 0 24 24', path: 'M12 2a10 10 0 1 0 0 20c.8 0 1.5-.7 1.5-1.5 0-.4-.1-.7-.4-1-.3-.3-.4-.7-.4-1.1 0-.8.7-1.5 1.5-1.5H16a6 6 0 0 0 6-6c0-4.4-3.9-8.9-10-8.9zm-1.5 5.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3zm-5 5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3zm8 2a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z' },
}

type IconName = keyof typeof icons

export default function Icon({ name, size = 18 }: { name: IconName; size?: number }) {
  const icon = icons[name]
  return (
    <svg
      width={size}
      height={size}
      viewBox={icon.viewBox}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d={icon.path} />
    </svg>
  )
}
