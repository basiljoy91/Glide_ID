import { redirect } from 'next/navigation'

export default function Home() {
  // For now, show landing page
  // Later: Check auth and redirect accordingly
  redirect('/landing')
}

