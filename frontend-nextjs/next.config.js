/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  images: {
    domains: [],
  },
  // PWA Configuration
  experimental: {
    appDir: true,
  },
}

module.exports = nextConfig

