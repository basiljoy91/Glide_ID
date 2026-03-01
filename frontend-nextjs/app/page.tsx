import { redirect } from 'next/navigation'

export default function Home() {
  // Redirect to appropriate page based on authentication
  redirect('/dashboard')
}

