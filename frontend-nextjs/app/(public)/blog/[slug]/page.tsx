import Link from 'next/link'
import type { Metadata } from 'next'
import { blogPosts } from '@/lib/blogData'
import { Button } from '@/components/ui/button'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

type Params = { params: { slug: string } }

export function generateMetadata({ params }: Params): Metadata {
  const post = blogPosts.find((p) => p.slug === params.slug)
  if (!post) {
    return {
      title: 'Article not found | Glide ID Journal',
      metadataBase: new URL(siteUrl),
      alternates: { canonical: `/blog/${params.slug}` },
    }
  }

  return {
    title: `${post.title} | Glide ID Journal`,
    description: post.excerpt,
    metadataBase: new URL(siteUrl),
    alternates: { canonical: `/blog/${post.slug}` },
    openGraph: {
      title: post.title,
      description: post.excerpt,
      url: `/blog/${post.slug}`,
      siteName: 'Glide ID',
      images: [{ url: '/hero-visual.svg', width: 1200, height: 630, alt: post.title }],
      type: 'article',
    },
    twitter: {
      card: 'summary_large_image',
      title: post.title,
      description: post.excerpt,
      images: ['/hero-visual.svg'],
    },
  }
}

export default function BlogPostPage({ params }: Params) {
  const post = blogPosts.find((p) => p.slug === params.slug)

  if (!post) {
    return (
      <div className="container mx-auto px-4 py-16">
        <div className="rounded-2xl border bg-background/80 p-6">
          <h1 className="text-2xl font-display font-semibold">Article not found</h1>
          <p className="mt-3 text-sm text-muted-foreground">
            The article you requested does not exist.
          </p>
          <div className="mt-6">
            <Link href="/blog">
              <Button variant="outline">Back to Journal</Button>
            </Link>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="space-y-3">
        <div className="text-xs uppercase tracking-[0.25em] text-muted-foreground">
          {post.category}
        </div>
        <h1 className="text-4xl font-display font-semibold">{post.title}</h1>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span>{post.date}</span>
          <span>{post.readTime}</span>
        </div>
      </div>

      <div className="rounded-3xl border bg-background/80 p-8 space-y-4 text-muted-foreground">
        {post.content.map((para) => (
          <p key={para}>{para}</p>
        ))}
      </div>

      <div className="flex gap-3">
        <Link href="/blog">
          <Button variant="outline">Back to Journal</Button>
        </Link>
        <Link href="/contact">
          <Button>Talk to the team</Button>
        </Link>
      </div>
    </div>
  )
}
