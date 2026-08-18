import { Redirect } from 'expo-router'
import { useEffect, useState } from 'react'

import { hasSession } from '@/services/session'

export default function Index() {
  const [destination, setDestination] = useState<'/(app)' | '/(auth)/login'>()

  useEffect(() => {
    hasSession().then((active) => setDestination(active ? '/(app)' : '/(auth)/login'))
  }, [])

  if (!destination) return null
  return <Redirect href={destination} />
}
