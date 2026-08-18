import { Redirect, Stack } from 'expo-router'
import { useEffect, useState } from 'react'

import { hasSession } from '@/services/session'

export default function AuthenticatedLayout() {
  const [authenticated, setAuthenticated] = useState<boolean>()

  useEffect(() => {
    hasSession().then(setAuthenticated)
  }, [])

  if (authenticated === undefined) return null
  if (!authenticated) return <Redirect href="/(auth)/login" />
  return <Stack screenOptions={{ headerShown: false }} />
}
