/** @type {import('next').NextConfig} */
const nextConfig = {
    output: 'standalone',
    reactStrictMode: true,
    async rewrites() {
        // NEXT_PUBLIC_API_URL is set by Docker to http://backend:8080/api/v1
        // For local dev, fallback to localhost:8080
        const backendUrl = process.env.NEXT_PUBLIC_API_URL
            ? process.env.NEXT_PUBLIC_API_URL.replace('/api/v1', '')  // Remove /api/v1 suffix if present
            : 'http://localhost:8080';

        return [
            {
                source: '/api/:path*',
                destination: `${backendUrl}/api/:path*`,
            },
        ]
    },
}

module.exports = nextConfig
