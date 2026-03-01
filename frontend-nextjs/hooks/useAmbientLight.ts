'use client'

import { useState, useEffect } from 'react'

export function useAmbientLight() {
  const [isDark, setIsDark] = useState(false)
  const [brightness, setBrightness] = useState(1)

  useEffect(() => {
    // Check if Ambient Light Sensor API is available
    if ('AmbientLightSensor' in window) {
      // @ts-ignore - AmbientLightSensor is not in TypeScript types yet
      const sensor = new AmbientLightSensor()
      
      sensor.addEventListener('reading', () => {
        const illuminance = sensor.illuminance
        // Consider dark if illuminance is less than 10 lux
        setIsDark(illuminance < 10)
        // Adjust brightness based on illuminance (normalize to 0-1)
        setBrightness(Math.min(1, illuminance / 100))
      })

      sensor.start()

      return () => {
        sensor.stop()
      }
    } else {
      // Fallback: Use system preference or time-based detection
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      setIsDark(mediaQuery.matches)
      
      const handleChange = (e: MediaQueryListEvent) => setIsDark(e.matches)
      mediaQuery.addEventListener('change', handleChange)
      
      return () => mediaQuery.removeEventListener('change', handleChange)
    }
  }, [])

  return { isDark, brightness }
}

